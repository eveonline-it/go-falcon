package sde

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	pkgsde "go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
	goredis "github.com/redis/go-redis/v9"
)

const (
	sdeURL          = "https://eve-static-data-export.s3-eu-west-1.amazonaws.com/tranquility/sde.zip"
	sdeHashURL      = "https://eve-static-data-export.s3-eu-west-1.amazonaws.com/tranquility/checksum"
	redisHashKey    = "sde:current_hash"
	redisStatusKey  = "sde:status"
	redisProgressKey = "sde:progress"
	solarSystemIndexKey = "sde:index:solarsystems"
)

// GroupsModule interface defines the methods needed from the groups module
type GroupsModule interface {
	RequireGranularPermission(service, resource, action string) func(http.Handler) http.Handler
}

type Module struct {
	*module.BaseModule
	mu              sync.RWMutex
	isProcessing    bool
	currentProgress float64
	lastError       error
	lastCheck       time.Time
	groupsModule    GroupsModule
}

type SDEStatus struct {
	CurrentHash     string    `json:"current_hash"`
	LatestHash      string    `json:"latest_hash"`
	IsUpToDate      bool      `json:"is_up_to_date"`
	IsProcessing    bool      `json:"is_processing"`
	Progress        float64   `json:"progress"`
	LastError       string    `json:"last_error,omitempty"`
	LastCheck       time.Time `json:"last_check"`
	LastUpdate      time.Time `json:"last_update"`
}

type UpdateRequest struct {
	ForceUpdate bool `json:"force_update"`
}

func New(mongodb *database.MongoDB, redis *database.Redis, sdeService pkgsde.SDEService, groupsModule GroupsModule) *Module {
	return &Module{
		BaseModule:   module.NewBaseModule("sde", mongodb, redis, sdeService),
		groupsModule: groupsModule,
	}
}

func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r) // Use the base module health handler
	
	// Public read-only endpoints
	r.Get("/status", m.handleGetStatus)
	r.Get("/progress", m.handleGetProgress)
	
	// Protected data access endpoints
	r.With(m.groupsModule.RequireGranularPermission("sde", "entities", "read")).Get("/entity/{type}/{id}", m.handleGetEntity)
	r.With(m.groupsModule.RequireGranularPermission("sde", "entities", "read")).Get("/entities/{type}", m.handleGetEntitiesByType)
	r.With(m.groupsModule.RequireGranularPermission("sde", "entities", "read")).Get("/search/solarsystem", m.handleSearchSolarSystem)
	
	// Protected management endpoints
	r.With(m.groupsModule.RequireGranularPermission("sde", "entities", "write")).Post("/check", m.handleCheckForUpdates)
	r.With(m.groupsModule.RequireGranularPermission("sde", "entities", "admin")).Post("/update", m.handleStartUpdate)
	r.With(m.groupsModule.RequireGranularPermission("sde", "entities", "admin")).Post("/index/rebuild", m.handleRebuildIndex)
	
	// Development/testing endpoints (admin only)
	r.With(m.groupsModule.RequireGranularPermission("sde", "entities", "admin")).Post("/test/store-sample", m.handleTestStoreSample)
	r.With(m.groupsModule.RequireGranularPermission("sde", "entities", "admin")).Get("/test/verify", m.handleTestVerify)
}

func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting SDE background tasks")
	
	// Load current status from Redis
	if err := m.loadStatus(); err != nil {
		slog.Warn("Failed to load SDE status from Redis", "error", err)
	}
}

func (m *Module) Stop() {
	slog.Info("Stopping SDE module")
}

// handleGetStatus returns the current SDE status
func (m *Module) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	status, err := m.getStatus(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get SDE status: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleCheckForUpdates checks if a new SDE version is available
func (m *Module) handleCheckForUpdates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	status, err := m.checkForUpdates(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check for updates: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleStartUpdate initiates the SDE update process
func (m *Module) handleStartUpdate(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	
	m.mu.Lock()
	if m.isProcessing {
		m.mu.Unlock()
		http.Error(w, "Update already in progress", http.StatusConflict)
		return
	}
	m.isProcessing = true
	m.currentProgress = 0
	m.lastError = nil
	m.mu.Unlock()
	
	// Start update in background
	go m.processSDEUpdate(context.Background(), req.ForceUpdate)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "SDE update started",
		"status":  "processing",
	})
}

// handleGetProgress returns the current update progress
func (m *Module) handleGetProgress(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	progress := map[string]interface{}{
		"is_processing": m.isProcessing,
		"progress":      m.currentProgress,
	}
	if m.lastError != nil {
		progress["error"] = m.lastError.Error()
	}
	m.mu.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(progress)
}

// getStatus retrieves the current SDE status
func (m *Module) getStatus(ctx context.Context) (*SDEStatus, error) {
	redis := m.Redis()
	
	// Get current hash from Redis
	currentHash, err := redis.Client.Get(ctx, redisHashKey).Result()
	if err != nil && err.Error() != "redis: nil" {
		return nil, fmt.Errorf("failed to get current hash: %w", err)
	}
	
	// Get stored status
	statusJSON, err := redis.Client.Get(ctx, redisStatusKey).Result()
	if err != nil && err.Error() != "redis: nil" {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}
	
	var status SDEStatus
	if statusJSON != "" {
		if err := json.Unmarshal([]byte(statusJSON), &status); err != nil {
			return nil, fmt.Errorf("failed to unmarshal status: %w", err)
		}
	}
	
	status.CurrentHash = currentHash
	
	m.mu.RLock()
	status.IsProcessing = m.isProcessing
	status.Progress = m.currentProgress
	if m.lastError != nil {
		status.LastError = m.lastError.Error()
	}
	status.LastCheck = m.lastCheck
	m.mu.RUnlock()
	
	return &status, nil
}

// checkForUpdates checks if a new SDE version is available
func (m *Module) checkForUpdates(ctx context.Context) (*SDEStatus, error) {
	// Fetch latest hash from CCP
	latestHash, err := m.fetchLatestHash()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest hash: %w", err)
	}
	
	// Get current status
	status, err := m.getStatus(ctx)
	if err != nil {
		return nil, err
	}
	
	status.LatestHash = latestHash
	status.IsUpToDate = (status.CurrentHash == latestHash)
	
	m.mu.Lock()
	m.lastCheck = time.Now()
	m.mu.Unlock()
	
	// Save status to Redis
	if err := m.saveStatus(ctx, status); err != nil {
		slog.Warn("Failed to save status to Redis", "error", err)
	}
	
	// Send notification if update is available
	if !status.IsUpToDate {
		m.sendUpdateNotification(status)
	}
	
	return status, nil
}

// fetchLatestHash fetches the latest SDE hash from CCP
func (m *Module) fetchLatestHash() (string, error) {
	resp, err := http.Get(sdeHashURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch hash: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	hashBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read hash: %w", err)
	}
	
	// The checksum file contains the MD5 hash
	return string(hashBytes[:32]), nil
}

// processSDEUpdate performs the complete SDE update process
func (m *Module) processSDEUpdate(ctx context.Context, forceUpdate bool) {
	defer func() {
		m.mu.Lock()
		m.isProcessing = false
		m.mu.Unlock()
	}()
	
	// Check if update is needed
	if !forceUpdate {
		status, err := m.checkForUpdates(ctx)
		if err != nil {
			m.setError(fmt.Errorf("failed to check for updates: %w", err))
			return
		}
		
		if status.IsUpToDate {
			slog.Info("SDE is already up to date")
			return
		}
	}
	
	// Update progress
	m.updateProgress(0.1, "Downloading SDE file...")
	
	// Download SDE file
	tmpDir := filepath.Join(os.TempDir(), "sde_update")
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		m.setError(fmt.Errorf("failed to create temp directory: %w", err))
		return
	}
	defer os.RemoveAll(tmpDir)
	
	sdeZipFile := filepath.Join(tmpDir, "sde.zip")
	if err := m.downloadFile(sdeZipFile, sdeURL); err != nil {
		m.setError(fmt.Errorf("failed to download SDE: %w", err))
		return
	}
	
	// Update progress
	m.updateProgress(0.3, "Extracting SDE file...")
	
	// Extract SDE file
	extractDir := filepath.Join(tmpDir, "sde")
	if err := m.unzipFile(sdeZipFile, extractDir); err != nil {
		m.setError(fmt.Errorf("failed to extract SDE: %w", err))
		return
	}
	
	// Update progress
	m.updateProgress(0.5, "Converting YAML to JSON...")
	
	// Convert YAML files to JSON
	if err := m.convertYAMLFiles(extractDir); err != nil {
		m.setError(fmt.Errorf("failed to convert YAML files: %w", err))
		return
	}
	
	// Update progress
	m.updateProgress(0.7, "Storing data in Redis...")
	
	// Clean up old SDE data first
	if err := m.CleanupOldSDEData(ctx); err != nil {
		slog.Warn("Failed to cleanup old SDE data", "error", err)
	}
	
	// Store data in Redis
	if err := m.storeInRedis(ctx); err != nil {
		m.setError(fmt.Errorf("failed to store data in Redis: %w", err))
		return
	}
	
	// Build solar system search index
	m.updateProgress(0.8, "Building search indexes...")
	if err := m.buildSolarSystemIndex(ctx); err != nil {
		slog.Warn("Failed to build solar system index", "error", err)
		// Don't fail the entire update for index building
	}
	
	// Update progress
	m.updateProgress(0.9, "Finalizing update...")
	
	// Calculate and store new hash
	hash := m.calculateFileHash(sdeZipFile)
	redis := m.Redis()
	if err := redis.Client.Set(ctx, redisHashKey, hash, 0).Err(); err != nil {
		m.setError(fmt.Errorf("failed to store hash: %w", err))
		return
	}
	
	// Update status
	status := &SDEStatus{
		CurrentHash: hash,
		LatestHash:  hash,
		IsUpToDate:  true,
		LastUpdate:  time.Now(),
		LastCheck:   time.Now(),
	}
	
	if err := m.saveStatus(ctx, status); err != nil {
		m.setError(fmt.Errorf("failed to save status: %w", err))
		return
	}
	
	// Update progress
	m.updateProgress(1.0, "Update completed successfully")
	
	slog.Info("SDE update completed successfully", "hash", hash)
	
	// Send completion notification
	m.sendCompletionNotification(status)
}

// downloadFile downloads a file from a URL with progress tracking
func (m *Module) downloadFile(filepath string, url string) error {
	return m.downloadFileWithProgress(filepath, url)
}


// convertYAMLFiles converts all YAML files from bsd, fsd, and universe directories to JSON
func (m *Module) convertYAMLFiles(extractDir string) error {
	jsonDir := "data/sde"
	
	// Standard directories to scan for YAML files
	scanDirs := []string{"bsd", "fsd"}
	
	// Collect all YAML files from standard directories
	var yamlFiles []string
	for _, dir := range scanDirs {
		dirPath := filepath.Join(extractDir, dir)
		files, err := collectYAMLFiles(dirPath)
		if err != nil {
			slog.Warn("Failed to collect YAML files from directory", "dir", dirPath, "error", err)
			continue
		}
		yamlFiles = append(yamlFiles, files...)
	}
	
	// Process universe directory with special handling
	universeFiles, err := m.collectUniverseYAMLFiles(extractDir)
	if err != nil {
		slog.Warn("Failed to collect universe YAML files", "error", err)
	} else {
		yamlFiles = append(yamlFiles, universeFiles...)
	}
	
	if len(yamlFiles) == 0 {
		return fmt.Errorf("no YAML files found in bsd, fsd, or universe directories")
	}
	
	slog.Info("Found YAML files to process", "count", len(yamlFiles))
	totalFiles := len(yamlFiles)
	
	for i, yamlFile := range yamlFiles {
		fullPath := filepath.Join(extractDir, yamlFile)
		
		// Update progress
		progress := 0.5 + (0.2 * float64(i) / float64(totalFiles))
		m.updateProgress(progress, fmt.Sprintf("Converting %s... (%d/%d)", filepath.Base(yamlFile), i+1, totalFiles))
		
		if err := convertYAMLToJSON(fullPath, jsonDir); err != nil {
			slog.Error("Failed to convert YAML file", "file", fullPath, "error", err)
			// Continue with other files even if one fails
			continue
		}
		
		slog.Info("Converted YAML file", "file", fullPath)
	}
	
	return nil
}

// collectUniverseYAMLFiles collects YAML files from universe directory with hierarchical structure
// Structure: universe/{type}/{region}/{constellation}/{system}/
// Processes: abyssal, eve, hidden, void, and wormhole subdirectories
// Excludes: landmarks directory
func (m *Module) collectUniverseYAMLFiles(extractDir string) ([]string, error) {
	universeDir := filepath.Join(extractDir, "universe")
	
	// Check if universe directory exists
	if _, err := os.Stat(universeDir); os.IsNotExist(err) {
		return []string{}, nil // Return empty list if universe directory doesn't exist
	}
	
	// Allowed universe subdirectories
	allowedDirs := []string{"abyssal", "eve", "hidden", "void", "wormhole"}
	var universeFiles []string
	
	for _, universeType := range allowedDirs {
		universeTypeDir := filepath.Join(universeDir, universeType)
		
		// Check if universe type directory exists
		if _, err := os.Stat(universeTypeDir); os.IsNotExist(err) {
			slog.Debug("Universe type directory not found", "type", universeType)
			continue
		}
		
		// Collect files from this universe type with hierarchical structure
		typeFiles, err := m.collectUniverseTypeFiles(universeTypeDir, universeType)
		if err != nil {
			slog.Warn("Failed to collect universe type files", "type", universeType, "error", err)
			continue
		}
		
		universeFiles = append(universeFiles, typeFiles...)
		slog.Info("Collected universe type files", "type", universeType, "count", len(typeFiles))
	}
	
	slog.Info("Total universe YAML files collected", "count", len(universeFiles))
	return universeFiles, nil
}

// collectUniverseTypeFiles collects YAML files from a universe type directory
// Handles hierarchical structure: {type}/{region}/{constellation}/{system}/
func (m *Module) collectUniverseTypeFiles(universeTypeDir, universeType string) ([]string, error) {
	var files []string
	
	// Walk through the hierarchical structure
	err := filepath.Walk(universeTypeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Check for YAML files
		ext := filepath.Ext(path)
		if ext == ".yaml" || ext == ".yml" {
			// Get relative path from the extract directory root
			relPath, err := filepath.Rel(filepath.Dir(filepath.Dir(universeTypeDir)), path)
			if err != nil {
				slog.Warn("Failed to get relative path", "path", path, "error", err)
				return nil // Continue processing other files
			}
			
			files = append(files, relPath)
			
			// Log hierarchical info for debugging
			pathParts := strings.Split(strings.TrimPrefix(relPath, "universe/"), string(filepath.Separator))
			if len(pathParts) >= 4 {
				slog.Debug("Found universe YAML file", 
					"type", pathParts[0], 
					"region", pathParts[1], 
					"constellation", pathParts[2], 
					"system", pathParts[3],
					"file", filepath.Base(path))
			}
		}
		
		return nil
	})
	
	return files, err
}

// storeInRedis stores SDE data in Redis as individual JSON entries
func (m *Module) storeInRedis(ctx context.Context) error {
	jsonDir := "data/sde"
	
	files, err := os.ReadDir(jsonDir)
	if err != nil {
		return fmt.Errorf("failed to read JSON directory: %w", err)
	}
	
	// Process each SDE file type
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}
		
		filePath := filepath.Join(jsonDir, file.Name())
		baseName := file.Name()[:len(file.Name())-5] // Remove .json
		
		slog.Info("Processing SDE file for individual storage", "file", baseName)
		
		// Determine data type and context from file path
		dataType, pathContext := m.determineDataTypeAndContext(baseName, filePath)
		
		if err := m.storeSDEFileAsIndividualKeysWithContext(ctx, filePath, dataType, pathContext); err != nil {
			return fmt.Errorf("failed to store %s as individual keys: %w", baseName, err)
		}
	}
	
	return nil
}

// determineDataTypeAndContext determines the data type and hierarchical context from file path
func (m *Module) determineDataTypeAndContext(baseName, filePath string) (string, map[string]string) {
	pathContext := make(map[string]string)
	
	// Check if this is a universe file by examining the base name
	if strings.Contains(baseName, "universe") {
		// Parse universe path: universe_{type}_{region}_{constellation}_{system}_{filename}
		// or try to extract from the original YAML path structure
		parts := strings.Split(baseName, "_")
		if len(parts) >= 2 && parts[0] == "universe" {
			pathContext["universe_type"] = parts[1]
			if len(parts) >= 3 {
				pathContext["region"] = parts[2]
			}
			if len(parts) >= 4 {
				pathContext["constellation"] = parts[3]
			}
			if len(parts) >= 5 {
				pathContext["system"] = parts[4]
			}
			
			// Determine what kind of universe data this is based on filename
			fileName := parts[len(parts)-1]
			if strings.Contains(fileName, "region") {
				return "universe_regions", pathContext
			} else if strings.Contains(fileName, "constellation") {
				return "universe_constellations", pathContext
			} else if strings.Contains(fileName, "system") || strings.Contains(fileName, "solarsystem") {
				return "universe_systems", pathContext
			} else {
				return "universe_objects", pathContext
			}
		}
	}
	
	// For non-universe files, use the base name as data type
	return baseName, pathContext
}

// storeSDEFileAsIndividualKeysWithContext stores SDE data with hierarchical context awareness
func (m *Module) storeSDEFileAsIndividualKeysWithContext(ctx context.Context, filePath, dataType string, pathContext map[string]string) error {
	// For backwards compatibility, if no context, use the original method
	if len(pathContext) == 0 {
		return m.storeSDEFileAsIndividualKeys(ctx, filePath, dataType)
	}
	
	redisClient := m.Redis()
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	
	// For universe files, store the complete JSON as a single Redis key
	if _, isUniverse := pathContext["universe_type"]; isUniverse {
		key := m.generateUniverseRedisKey(dataType, "", pathContext)
		
		// Store the complete JSON file content
		err = redisClient.Client.Do(ctx, "JSON.SET", key, "$", data).Err()
		if err != nil {
			return fmt.Errorf("failed to store universe file %s: %w", key, err)
		}
		
		slog.Info("Stored complete universe file", 
			"key", key, 
			"size", len(data),
			"context", pathContext)
		
		return nil
	}
	
	// For non-universe files with context, fall back to individual entity storage
	return m.storeSDEFileAsIndividualKeys(ctx, filePath, dataType)
}

// generateUniverseRedisKey generates Redis keys for universe data with hierarchical context
func (m *Module) generateUniverseRedisKey(dataType, entityID string, pathContext map[string]string) string {
	// For universe data, include hierarchical information in the key
	if universeType, exists := pathContext["universe_type"]; exists {
		if region, hasRegion := pathContext["region"]; hasRegion {
			if constellation, hasConstellation := pathContext["constellation"]; hasConstellation {
				if system, hasSystem := pathContext["system"]; hasSystem {
					// Full path for complete file: sde:universe:{type}:{region}:{constellation}:{system}
					if entityID == "" {
						return fmt.Sprintf("sde:universe:%s:%s:%s:%s", universeType, region, constellation, system)
					}
					// Individual entity: sde:universe:{type}:{region}:{constellation}:{system}:{entityID}
					return fmt.Sprintf("sde:universe:%s:%s:%s:%s:%s", universeType, region, constellation, system, entityID)
				}
				// Constellation level
				if entityID == "" {
					return fmt.Sprintf("sde:universe:%s:%s:%s", universeType, region, constellation)
				}
				return fmt.Sprintf("sde:universe:%s:%s:%s:%s", universeType, region, constellation, entityID)
			}
			// Region level
			if entityID == "" {
				return fmt.Sprintf("sde:universe:%s:%s", universeType, region)
			}
			return fmt.Sprintf("sde:universe:%s:%s:%s", universeType, region, entityID)
		}
		// Type level
		if entityID == "" {
			return fmt.Sprintf("sde:universe:%s", universeType)
		}
		return fmt.Sprintf("sde:universe:%s:%s", universeType, entityID)
	}
	
	// Fallback to standard format
	if entityID == "" {
		return fmt.Sprintf("sde:%s", dataType)
	}
	return fmt.Sprintf("sde:%s:%s", dataType, entityID)
}

// storeSDEFileAsIndividualKeys stores a single SDE file as individual Redis JSON entries
// Handles both map[string]interface{} and []interface{} formats
func (m *Module) storeSDEFileAsIndividualKeys(ctx context.Context, filePath, dataType string) error {
	redisClient := m.Redis()
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	
	// Parse JSON to determine structure
	var parsedData interface{}
	if err := json.Unmarshal(data, &parsedData); err != nil {
		return fmt.Errorf("failed to unmarshal %s: %w", filePath, err)
	}
	
	// Use Redis pipeline for batch operations
	pipe := redisClient.Client.Pipeline()
	storedCount := 0
	
	switch data := parsedData.(type) {
	case map[string]interface{}:
		// Handle map format (key-value pairs)
		for entityID, entityData := range data {
			key := fmt.Sprintf("sde:%s:%s", dataType, entityID)
			
			entityJSON, err := json.Marshal(entityData)
			if err != nil {
				slog.Warn("Failed to marshal entity", "type", dataType, "id", entityID, "error", err)
				continue
			}
			
			pipe.Do(ctx, "JSON.SET", key, "$", entityJSON)
			slog.Debug("Storing individual SDE entity", "key", key, "size", len(entityJSON))
			storedCount++
		}
		
	case []interface{}:
		// Handle array format (list of objects)
		for i, item := range data {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				slog.Warn("Array item is not an object", "type", dataType, "index", i)
				continue
			}
			
			// Try to find a suitable ID field
			entityID := m.extractEntityID(itemMap, i)
			key := fmt.Sprintf("sde:%s:%s", dataType, entityID)
			
			entityJSON, err := json.Marshal(itemMap)
			if err != nil {
				slog.Warn("Failed to marshal array entity", "type", dataType, "id", entityID, "error", err)
				continue
			}
			
			pipe.Do(ctx, "JSON.SET", key, "$", entityJSON)
			slog.Debug("Storing individual SDE array entity", "key", key, "size", len(entityJSON))
			storedCount++
		}
		
	default:
		return fmt.Errorf("unsupported data format for %s: expected map or array, got %T", dataType, parsedData)
	}
	
	if storedCount == 0 {
		slog.Warn("No entities stored", "type", dataType, "file", filePath)
		return nil
	}
	
	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute Redis pipeline for %s: %w", dataType, err)
	}
	
	slog.Info("Stored SDE data type as individual keys", 
		"type", dataType, 
		"count", storedCount,
		"pattern", fmt.Sprintf("sde:%s:*", dataType))
	
	return nil
}

// extractEntityID attempts to extract a suitable ID from an entity object
// For array-based data, tries common ID fields or falls back to index
func (m *Module) extractEntityID(entity map[string]interface{}, fallbackIndex int) string {
	// Common ID field names in EVE SDE data
	idFields := []string{
		"id", "ID", "itemID", "typeID", "flagID", "groupID", 
		"categoryID", "marketGroupID", "corporationID", "factionID",
		"agentID", "stationID", "systemID", "regionID", "blueprintID",
		"materialTypeID", "activityID", "raceID", "bloodlineID",
		"ancestryID", "attributeID", "unitID", "iconID", "graphicID",
		"constellationID", "solarSystemID", "planetID", "moonID", "starID",
		"belt", "asteroid", "gate", "deadspaceID", "wormholeID",
	}
	
	// Try to find a suitable ID field
	for _, field := range idFields {
		if value, exists := entity[field]; exists {
			// Convert to string
			switch v := value.(type) {
			case int:
				return fmt.Sprintf("%d", v)
			case int64:
				return fmt.Sprintf("%d", v)
			case float64:
				return fmt.Sprintf("%.0f", v)
			case string:
				if v != "" {
					return v
				}
			}
		}
	}
	
	// If no ID field found, try using the first string/number field as identifier
	for key, value := range entity {
		switch v := value.(type) {
		case int:
			return fmt.Sprintf("%s_%d", key, v)
		case int64:
			return fmt.Sprintf("%s_%d", key, v)
		case float64:
			return fmt.Sprintf("%s_%.0f", key, v)
		case string:
			if v != "" && len(v) < 50 { // Reasonable length for an ID
				return fmt.Sprintf("%s_%s", key, v)
			}
		}
	}
	
	// Last resort: use array index
	return fmt.Sprintf("index_%d", fallbackIndex)
}

// calculateFileHash calculates MD5 hash of a file
func (m *Module) calculateFileHash(filepath string) string {
	file, err := os.Open(filepath)
	if err != nil {
		return ""
	}
	defer file.Close()
	
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}
	
	return hex.EncodeToString(hash.Sum(nil))
}

// updateProgress updates the current progress
func (m *Module) updateProgress(progress float64, message string) {
	m.mu.Lock()
	m.currentProgress = progress
	m.mu.Unlock()
	
	// Store in Redis for distributed access
	redis := m.Redis()
	progressData := map[string]interface{}{
		"progress": progress,
		"message":  message,
		"time":     time.Now(),
	}
	
	data, _ := json.Marshal(progressData)
	redis.Client.Set(context.Background(), redisProgressKey, data, 1*time.Hour)
	
	slog.Info("SDE update progress", "progress", progress, "message", message)
}

// setError sets the last error
func (m *Module) setError(err error) {
	m.mu.Lock()
	m.lastError = err
	m.mu.Unlock()
	
	slog.Error("SDE update error", "error", err)
}

// saveStatus saves the status to Redis
func (m *Module) saveStatus(ctx context.Context, status *SDEStatus) error {
	redis := m.Redis()
	
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}
	
	return redis.Client.Set(ctx, redisStatusKey, data, 0).Err()
}

// loadStatus loads the status from Redis
func (m *Module) loadStatus() error {
	ctx := context.Background()
	redis := m.Redis()
	
	statusJSON, err := redis.Client.Get(ctx, redisStatusKey).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return nil // No status stored yet
		}
		return fmt.Errorf("failed to get status: %w", err)
	}
	
	var status SDEStatus
	if err := json.Unmarshal([]byte(statusJSON), &status); err != nil {
		return fmt.Errorf("failed to unmarshal status: %w", err)
	}
	
	m.mu.Lock()
	m.lastCheck = status.LastCheck
	m.mu.Unlock()
	
	return nil
}

// sendUpdateNotification sends a notification when an update is available
func (m *Module) sendUpdateNotification(status *SDEStatus) {
	// TODO: Implement notification logic
	// This could send to a notification service, webhook, or WebSocket
	slog.Info("New SDE version available", 
		"current", status.CurrentHash,
		"latest", status.LatestHash)
}

// sendCompletionNotification sends a notification when update is complete
func (m *Module) sendCompletionNotification(status *SDEStatus) {
	// TODO: Implement notification logic
	slog.Info("SDE update completed",
		"hash", status.CurrentHash,
		"time", status.LastUpdate)
}

// CheckSDEUpdate is called by the scheduler for periodic checks
func (m *Module) CheckSDEUpdate(ctx context.Context) error {
	status, err := m.checkForUpdates(ctx)
	if err != nil {
		return fmt.Errorf("failed to check SDE updates: %w", err)
	}
	
	if !status.IsUpToDate {
		slog.Info("New SDE version available, starting automatic update")
		go m.processSDEUpdate(context.Background(), false)
	}
	
	return nil
}

// GetSDEEntityFromRedis retrieves a single SDE entity from Redis by type and ID
func (m *Module) GetSDEEntityFromRedis(ctx context.Context, dataType, entityID string) (map[string]interface{}, error) {
	redis := m.Redis()
	key := fmt.Sprintf("sde:%s:%s", dataType, entityID)
	
	result, err := redis.Client.Do(ctx, "JSON.GET", key, "$").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get SDE entity %s: %w", key, err)
	}
	
	if result == nil {
		return nil, fmt.Errorf("SDE entity not found: %s", key)
	}
	
	// Redis JSON.GET returns a JSON string, parse it
	var entities []interface{}
	if err := json.Unmarshal([]byte(result.(string)), &entities); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SDE entity: %w", err)
	}
	
	if len(entities) == 0 {
		return nil, fmt.Errorf("SDE entity not found: %s", key)
	}
	
	entity, ok := entities[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid SDE entity format: %s", key)
	}
	
	return entity, nil
}

// GetSDEEntitiesByType retrieves all entities of a specific type from Redis
func (m *Module) GetSDEEntitiesByType(ctx context.Context, dataType string) (map[string]interface{}, error) {
	redis := m.Redis()
	pattern := fmt.Sprintf("sde:%s:*", dataType)
	
	// Get all keys matching the pattern
	keys, err := redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys for pattern %s: %w", pattern, err)
	}
	
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}
	
	// Use pipeline to get all entities
	pipe := redis.Client.Pipeline()
	for _, key := range keys {
		pipe.Do(ctx, "JSON.GET", key, "$")
	}
	
	results, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute pipeline for %s: %w", dataType, err)
	}
	
	entities := make(map[string]interface{})
	for i, result := range results {
		if result.Err() != nil {
			slog.Warn("Failed to get SDE entity", "key", keys[i], "error", result.Err())
			continue
		}
		
		// Extract entity ID from key (sde:type:id -> id)
		keyParts := strings.Split(keys[i], ":")
		if len(keyParts) != 3 {
			continue
		}
		entityID := keyParts[2]
		
		// Parse JSON result - handle the Redis command result  
		if result.Err() != nil {
			slog.Warn("Failed to get SDE entity", "key", keys[i], "error", result.Err())
			continue
		}
		
		// Cast to proper Redis command and get string result
		cmd := result.(*goredis.Cmd)
		resultStr, err := cmd.Text()
		if err != nil {
			slog.Warn("Failed to get text from SDE entity result", "key", keys[i], "error", err)
			continue
		}
		
		var entityArray []interface{}
		if err := json.Unmarshal([]byte(resultStr), &entityArray); err != nil {
			slog.Warn("Failed to unmarshal SDE entity", "key", keys[i], "error", err)
			continue
		}
		
		if len(entityArray) > 0 {
			entities[entityID] = entityArray[0]
		}
	}
	
	return entities, nil
}

// searchSolarSystemsByName searches for solar systems by name using fast index
func (m *Module) searchSolarSystemsByName(ctx context.Context, query string) ([]map[string]interface{}, error) {
	// Use fast indexed search
	return m.searchSolarSystemsByNameFast(ctx, query)
}

// searchSolarSystemsByNameOriginal is the original implementation (moved to slow)
func (m *Module) searchSolarSystemsByNameOriginal(ctx context.Context, query string) ([]map[string]interface{}, error) {
	redis := m.Redis()
	
	// Get all universe keys that represent solar systems
	pattern := "sde:universe:*"
	keys, err := redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get universe keys: %w", err)
	}
	
	var results []map[string]interface{}
	query = strings.ToLower(query)
	
	// Use pipeline to get multiple systems efficiently
	pipe := redis.Client.Pipeline()
	keyCommands := make(map[string]*goredis.Cmd)
	
	for _, key := range keys {
		// Only process keys that look like solar systems (not constellation.yaml)
		if strings.Contains(key, "constellation.yaml") || strings.Contains(key, "region.yaml") {
			continue
		}
		
		keyCommands[key] = pipe.Do(ctx, "JSON.GET", key, "$")
	}
	
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute pipeline: %w", err)
	}
	
	// Process results
	for key, cmd := range keyCommands {
		if cmd.Err() != nil {
			continue
		}
		
		resultStr, err := cmd.Text()
		if err != nil {
			continue
		}
		
		var systemArray []interface{}
		if err := json.Unmarshal([]byte(resultStr), &systemArray); err != nil {
			continue
		}
		
		if len(systemArray) == 0 {
			continue
		}
		
		system, ok := systemArray[0].(map[string]interface{})
		if !ok {
			continue
		}
		
		// Extract solar system name from the key path
		// Format: sde:universe:{type}:{region}:{constellation}:{system}
		keyParts := strings.Split(key, ":")
		if len(keyParts) >= 6 {
			systemName := keyParts[len(keyParts)-1]
			
			// Check if system name matches query (case-insensitive partial match)
			if strings.Contains(strings.ToLower(systemName), query) {
				// Add system info to results
				result := map[string]interface{}{
					"systemName":     systemName,
					"region":         keyParts[3],
					"constellation":  keyParts[4],
					"universeType":   keyParts[2],
					"redisKey":       key,
				}
				
				// Add solar system ID if available
				if solarSystemID, exists := system["solarSystemID"]; exists {
					result["solarSystemID"] = solarSystemID
				}
				
				// Add security status if available
				if security, exists := system["security"]; exists {
					result["security"] = security
				}
				
				// Add solar system name ID if available
				if nameID, exists := system["solarSystemNameID"]; exists {
					result["solarSystemNameID"] = nameID
				}
				
				results = append(results, result)
			}
		}
	}
	
	return results, nil
}

// buildSolarSystemIndex creates a search index for solar systems to speed up name-based searches
func (m *Module) buildSolarSystemIndex(ctx context.Context) error {
	redis := m.Redis()
	
	// Get all universe solar system keys
	pattern := "sde:universe:*"
	keys, err := redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get universe keys: %w", err)
	}
	
	// Clear existing index
	redis.Client.Del(ctx, solarSystemIndexKey)
	
	// Build index with system name -> Redis key mapping
	pipe := redis.Client.Pipeline()
	indexedCount := 0
	
	for _, key := range keys {
		// Skip constellation and region files
		if strings.Contains(key, "constellation.yaml") || strings.Contains(key, "region.yaml") {
			continue
		}
		
		// Extract system name from key: sde:universe:{type}:{region}:{constellation}:{system}
		keyParts := strings.Split(key, ":")
		if len(keyParts) >= 6 {
			systemName := strings.ToLower(keyParts[len(keyParts)-1])
			
			// Store in Redis hash: field = system name, value = Redis key
			pipe.HSet(ctx, solarSystemIndexKey, systemName, key)
			indexedCount++
		}
	}
	
	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to build solar system index: %w", err)
	}
	
	slog.Info("Built solar system search index", "systems_indexed", indexedCount)
	return nil
}

// searchSolarSystemsByNameFast uses the Redis index for fast name-based searches
func (m *Module) searchSolarSystemsByNameFast(ctx context.Context, query string) ([]map[string]interface{}, error) {
	redis := m.Redis()
	query = strings.ToLower(query)
	
	// Get all system names from index
	systemNames, err := redis.Client.HKeys(ctx, solarSystemIndexKey).Result()
	if err != nil {
		// Fallback to slow search if index doesn't exist
		slog.Warn("Solar system index not found, falling back to slow search")
		return m.searchSolarSystemsByNameSlow(ctx, query)
	}
	
	// Find matching system names
	var matchingKeys []string
	for _, systemName := range systemNames {
		if strings.Contains(systemName, query) {
			// Get Redis key for this system
			redisKey, err := redis.Client.HGet(ctx, solarSystemIndexKey, systemName).Result()
			if err == nil {
				matchingKeys = append(matchingKeys, redisKey)
			}
		}
	}
	
	if len(matchingKeys) == 0 {
		return []map[string]interface{}{}, nil
	}
	
	// Fetch system data for matching keys
	return m.fetchSystemDataBatch(ctx, matchingKeys, query)
}

// fetchSystemDataBatch efficiently fetches system data for multiple keys
func (m *Module) fetchSystemDataBatch(ctx context.Context, keys []string, query string) ([]map[string]interface{}, error) {
	redis := m.Redis()
	
	// Use pipeline for batch fetching
	pipe := redis.Client.Pipeline()
	keyCommands := make(map[string]*goredis.Cmd)
	
	for _, key := range keys {
		keyCommands[key] = pipe.Do(ctx, "JSON.GET", key, "$")
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch system data: %w", err)
	}
	
	var results []map[string]interface{}
	
	// Process results
	for key, cmd := range keyCommands {
		if cmd.Err() != nil {
			continue
		}
		
		resultStr, err := cmd.Text()
		if err != nil {
			continue
		}
		
		var systemArray []interface{}
		if err := json.Unmarshal([]byte(resultStr), &systemArray); err != nil {
			continue
		}
		
		if len(systemArray) == 0 {
			continue
		}
		
		system, ok := systemArray[0].(map[string]interface{})
		if !ok {
			continue
		}
		
		// Extract system info from key
		keyParts := strings.Split(key, ":")
		if len(keyParts) >= 6 {
			systemName := keyParts[len(keyParts)-1]
			
			result := map[string]interface{}{
				"systemName":     systemName,
				"region":         keyParts[3],
				"constellation":  keyParts[4],
				"universeType":   keyParts[2],
				"redisKey":       key,
			}
			
			// Add system properties
			if solarSystemID, exists := system["solarSystemID"]; exists {
				result["solarSystemID"] = solarSystemID
			}
			if security, exists := system["security"]; exists {
				result["security"] = security
			}
			if nameID, exists := system["solarSystemNameID"]; exists {
				result["solarSystemNameID"] = nameID
			}
			
			results = append(results, result)
		}
	}
	
	return results, nil
}

// searchSolarSystemsByNameSlow is the original slow implementation (fallback)
func (m *Module) searchSolarSystemsByNameSlow(ctx context.Context, query string) ([]map[string]interface{}, error) {
	redis := m.Redis()
	
	// Get all universe keys that represent solar systems
	pattern := "sde:universe:*"
	keys, err := redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get universe keys: %w", err)
	}
	
	var results []map[string]interface{}
	query = strings.ToLower(query)
	
	// Use pipeline to get multiple systems efficiently
	pipe := redis.Client.Pipeline()
	keyCommands := make(map[string]*goredis.Cmd)
	
	for _, key := range keys {
		// Only process keys that look like solar systems (not constellation.yaml)
		if strings.Contains(key, "constellation.yaml") || strings.Contains(key, "region.yaml") {
			continue
		}
		
		keyCommands[key] = pipe.Do(ctx, "JSON.GET", key, "$")
	}
	
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute pipeline: %w", err)
	}
	
	// Process results
	for key, cmd := range keyCommands {
		if cmd.Err() != nil {
			continue
		}
		
		resultStr, err := cmd.Text()
		if err != nil {
			continue
		}
		
		var systemArray []interface{}
		if err := json.Unmarshal([]byte(resultStr), &systemArray); err != nil {
			continue
		}
		
		if len(systemArray) == 0 {
			continue
		}
		
		system, ok := systemArray[0].(map[string]interface{})
		if !ok {
			continue
		}
		
		// Extract solar system name from the key path
		// Format: sde:universe:{type}:{region}:{constellation}:{system}
		keyParts := strings.Split(key, ":")
		if len(keyParts) >= 6 {
			systemName := keyParts[len(keyParts)-1]
			
			// Check if system name matches query (case-insensitive partial match)
			if strings.Contains(strings.ToLower(systemName), query) {
				// Add system info to results
				result := map[string]interface{}{
					"systemName":     systemName,
					"region":         keyParts[3],
					"constellation":  keyParts[4],
					"universeType":   keyParts[2],
					"redisKey":       key,
				}
				
				// Add solar system ID if available
				if solarSystemID, exists := system["solarSystemID"]; exists {
					result["solarSystemID"] = solarSystemID
				}
				
				// Add security status if available
				if security, exists := system["security"]; exists {
					result["security"] = security
				}
				
				// Add solar system name ID if available
				if nameID, exists := system["solarSystemNameID"]; exists {
					result["solarSystemNameID"] = nameID
				}
				
				results = append(results, result)
			}
		}
	}
	
	return results, nil
}

// CleanupOldSDEData removes all old SDE data keys before storing new ones
func (m *Module) CleanupOldSDEData(ctx context.Context) error {
	redis := m.Redis()
	
	// Get all existing SDE keys using pattern matching
	pattern := "sde:*"
	keys, err := redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get SDE keys for cleanup: %w", err)
	}
	
	if len(keys) == 0 {
		slog.Info("No existing SDE data to cleanup")
		return nil
	}
	
	// Filter out metadata keys that should not be deleted
	var dataKeys []string
	metadataKeys := map[string]bool{
		redisHashKey:        true,
		redisStatusKey:      true,
		redisProgressKey:    true,
		solarSystemIndexKey: true,
	}
	
	for _, key := range keys {
		if !metadataKeys[key] {
			dataKeys = append(dataKeys, key)
		}
	}
	
	if len(dataKeys) > 0 {
		// Delete data keys in batches to avoid large delete operations
		batchSize := 1000
		for i := 0; i < len(dataKeys); i += batchSize {
			end := i + batchSize
			if end > len(dataKeys) {
				end = len(dataKeys)
			}
			
			batch := dataKeys[i:end]
			deleted, err := redis.Client.Del(ctx, batch...).Result()
			if err != nil {
				slog.Warn("Failed to delete SDE data keys batch", "error", err, "batch_start", i)
			} else {
				slog.Info("Cleaned up SDE data keys batch", "deleted", deleted, "batch_start", i)
			}
		}
		
		slog.Info("Completed SDE data cleanup", "total_deleted_keys", len(dataKeys))
	}
	
	return nil
}

// handleGetEntity retrieves a single SDE entity by type and ID
func (m *Module) handleGetEntity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dataType := chi.URLParam(r, "type")
	entityID := chi.URLParam(r, "id")
	
	if dataType == "" || entityID == "" {
		http.Error(w, "Type and ID are required", http.StatusBadRequest)
		return
	}
	
	entity, err := m.GetSDEEntityFromRedis(ctx, dataType, entityID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, fmt.Sprintf("Entity not found: %s/%s", dataType, entityID), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to get entity: %v", err), http.StatusInternalServerError)
		}
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entity)
}

// handleGetEntitiesByType retrieves all entities of a specific type
func (m *Module) handleGetEntitiesByType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dataType := chi.URLParam(r, "type")
	
	if dataType == "" {
		http.Error(w, "Type is required", http.StatusBadRequest)
		return
	}
	
	entities, err := m.GetSDEEntitiesByType(ctx, dataType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get entities: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entities)
}

// handleSearchSolarSystem searches for solar systems by name
func (m *Module) handleSearchSolarSystem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get query parameter
	query := r.URL.Query().Get("name")
	if query == "" {
		http.Error(w, "Query parameter 'name' is required", http.StatusBadRequest)
		return
	}
	
	results, err := m.searchSolarSystemsByName(ctx, query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to search solar systems: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
}

// handleRebuildIndex manually rebuilds the solar system search index
func (m *Module) handleRebuildIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	start := time.Now()
	if err := m.buildSolarSystemIndex(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to rebuild index: %v", err), http.StatusInternalServerError)
		return
	}
	duration := time.Since(start)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Solar system index rebuilt successfully",
		"duration_ms": duration.Milliseconds(),
	})
}

// handleTestStoreSample stores some sample SDE data for testing individual key storage
func (m *Module) handleTestStoreSample(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	redis := m.Redis()
	
	// Sample test data
	testData := map[string]interface{}{
		"agents": map[string]interface{}{
			"3008416": map[string]interface{}{
				"agentTypeID":     2,
				"corporationID":   1000002,
				"divisionID":      1,
				"isLocator":       false,
				"level":           1,
				"locationID":      60000004,
				"quality":         0,
			},
			"3008417": map[string]interface{}{
				"agentTypeID":     2,
				"corporationID":   1000002,
				"divisionID":      1,
				"isLocator":       false,
				"level":           2,
				"locationID":      60000004,
				"quality":         10,
			},
		},
		"categories": map[string]interface{}{
			"1": map[string]interface{}{
				"name": map[string]interface{}{
					"en": "System",
				},
				"published": true,
			},
			"2": map[string]interface{}{
				"name": map[string]interface{}{
					"en": "Celestial",
				},
				"published": true,
			},
		},
	}
	
	// Store test data using individual keys
	pipe := redis.Client.Pipeline()
	stored := 0
	
	for dataType, entities := range testData {
		for entityID, entityData := range entities.(map[string]interface{}) {
			key := fmt.Sprintf("sde:%s:%s", dataType, entityID)
			entityJSON, _ := json.Marshal(entityData)
			pipe.Do(ctx, "JSON.SET", key, "$", entityJSON)
			stored++
		}
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to store test data: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Test data stored successfully",
		"stored":  stored,
		"types":   []string{"agents", "categories"},
	})
}

// handleTestVerify verifies that individual key storage works
func (m *Module) handleTestVerify(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Test individual entity retrieval
	agent, err := m.GetSDEEntityFromRedis(ctx, "agents", "3008416")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get test agent: %v", err), http.StatusInternalServerError)
		return
	}
	
	category, err := m.GetSDEEntityFromRedis(ctx, "categories", "1")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get test category: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Test bulk retrieval
	agents, err := m.GetSDEEntitiesByType(ctx, "agents")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get all agents: %v", err), http.StatusInternalServerError)
		return
	}
	
	categories, err := m.GetSDEEntitiesByType(ctx, "categories")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get all categories: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"individual_retrieval": map[string]interface{}{
			"agent_3008416": agent,
			"category_1":    category,
		},
		"bulk_retrieval": map[string]interface{}{
			"agents_count":     len(agents),
			"categories_count": len(categories),
		},
	})
}