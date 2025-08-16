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
)

type Module struct {
	*module.BaseModule
	mu              sync.RWMutex
	isProcessing    bool
	currentProgress float64
	lastError       error
	lastCheck       time.Time
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

func New(mongodb *database.MongoDB, redis *database.Redis, sdeService pkgsde.SDEService) *Module {
	return &Module{
		BaseModule: module.NewBaseModule("sde", mongodb, redis, sdeService),
	}
}

func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r) // Use the base module health handler
	r.Get("/status", m.handleGetStatus)
	r.Post("/check", m.handleCheckForUpdates)
	r.Post("/update", m.handleStartUpdate)
	r.Get("/progress", m.handleGetProgress)
	
	// Individual SDE entity access endpoints
	r.Get("/entity/{type}/{id}", m.handleGetEntity)
	r.Get("/entities/{type}", m.handleGetEntitiesByType)
	
	// Test endpoints for individual key storage
	r.Post("/test/store-sample", m.handleTestStoreSample)
	r.Get("/test/verify", m.handleTestVerify)
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


// convertYAMLFiles converts YAML files to JSON
func (m *Module) convertYAMLFiles(extractDir string) error {
	yamlFiles := []string{
		"fsd/agents.yaml",
		"fsd/blueprints.yaml",
		"fsd/categories.yaml",
		"fsd/marketGroups.yaml",
		"fsd/metaGroups.yaml",
		"fsd/npcCorporations.yaml",
		"fsd/types.yaml",
		"fsd/typeDogma.yaml",
		"fsd/typeMaterials.yaml",
	}
	
	jsonDir := "data/sde"
	totalFiles := len(yamlFiles)
	
	for i, yamlFile := range yamlFiles {
		fullPath := filepath.Join(extractDir, yamlFile)
		
		// Update progress
		progress := 0.5 + (0.2 * float64(i) / float64(totalFiles))
		m.updateProgress(progress, fmt.Sprintf("Converting %s...", filepath.Base(yamlFile)))
		
		if err := convertYAMLToJSON(fullPath, jsonDir); err != nil {
			slog.Error("Failed to convert YAML file", "file", fullPath, "error", err)
			// Continue with other files even if one fails
			continue
		}
		
		slog.Info("Converted YAML file", "file", fullPath)
	}
	
	return nil
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
		
		if err := m.storeSDEFileAsIndividualKeys(ctx, filePath, baseName); err != nil {
			return fmt.Errorf("failed to store %s as individual keys: %w", baseName, err)
		}
	}
	
	return nil
}

// storeSDEFileAsIndividualKeys stores a single SDE file as individual Redis JSON entries
func (m *Module) storeSDEFileAsIndividualKeys(ctx context.Context, filePath, dataType string) error {
	redisClient := m.Redis()
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	
	// Parse JSON as map to get individual entries
	var dataMap map[string]interface{}
	if err := json.Unmarshal(data, &dataMap); err != nil {
		return fmt.Errorf("failed to unmarshal %s: %w", filePath, err)
	}
	
	// Use Redis pipeline for batch operations
	pipe := redisClient.Client.Pipeline()
	
	// Store each entry as individual Redis JSON key
	for entityID, entityData := range dataMap {
		key := fmt.Sprintf("sde:%s:%s", dataType, entityID)
		
		// Convert entity data to JSON
		entityJSON, err := json.Marshal(entityData)
		if err != nil {
			slog.Warn("Failed to marshal entity", "type", dataType, "id", entityID, "error", err)
			continue
		}
		
		// Store as Redis JSON
		pipe.Do(ctx, "JSON.SET", key, "$", entityJSON)
		slog.Debug("Storing individual SDE entity", "key", key, "size", len(entityJSON))
	}
	
	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute Redis pipeline for %s: %w", dataType, err)
	}
	
	slog.Info("Stored SDE data type as individual keys", 
		"type", dataType, 
		"count", len(dataMap),
		"pattern", fmt.Sprintf("sde:%s:*", dataType))
	
	return nil
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

// CleanupOldSDEData removes old SDE data keys before storing new ones
func (m *Module) CleanupOldSDEData(ctx context.Context) error {
	redis := m.Redis()
	
	// SDE data types to clean
	dataTypes := []string{
		"agents", "categories", "blueprints", "marketGroups", 
		"metaGroups", "npcCorporations", "types", "typeDogma", "typeMaterials",
	}
	
	for _, dataType := range dataTypes {
		pattern := fmt.Sprintf("sde:%s:*", dataType)
		keys, err := redis.Client.Keys(ctx, pattern).Result()
		if err != nil {
			slog.Warn("Failed to get keys for cleanup", "pattern", pattern, "error", err)
			continue
		}
		
		if len(keys) > 0 {
			deleted, err := redis.Client.Del(ctx, keys...).Result()
			if err != nil {
				slog.Warn("Failed to delete old SDE keys", "pattern", pattern, "error", err)
			} else {
				slog.Info("Cleaned up old SDE data", "type", dataType, "deleted", deleted)
			}
		}
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