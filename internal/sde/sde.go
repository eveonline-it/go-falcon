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
	"sync"
	"time"

	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	pkgsde "go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
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

// storeInRedis stores SDE data in Redis
func (m *Module) storeInRedis(ctx context.Context) error {
	redis := m.Redis()
	jsonDir := "data/sde"
	
	files, err := os.ReadDir(jsonDir)
	if err != nil {
		return fmt.Errorf("failed to read JSON directory: %w", err)
	}
	
	pipe := redis.Client.Pipeline()
	
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}
		
		filePath := filepath.Join(jsonDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		
		// Store in Redis with appropriate key structure
		baseName := file.Name()[:len(file.Name())-5] // Remove .json
		key := fmt.Sprintf("sde:data:%s", baseName)
		
		pipe.Set(ctx, key, data, 0)
		slog.Debug("Storing SDE data in Redis", "key", key, "size", len(data))
	}
	
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute Redis pipeline: %w", err)
	}
	
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