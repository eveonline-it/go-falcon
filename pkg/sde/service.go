package sde

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// Service provides in-memory access to EVE Online SDE data
type Service struct {
	agents     map[string]*Agent
	categories map[string]*Category
	blueprints map[string]*Blueprint
	loaded     bool
	loadMu     sync.Mutex // Only used during initial loading
	dataDir    string
}

// NewService creates a new SDE service instance
func NewService(dataDir string) *Service {
	return &Service{
		agents:     make(map[string]*Agent),
		categories: make(map[string]*Category),
		blueprints: make(map[string]*Blueprint),
		dataDir:    dataDir,
	}
}

// ensureLoaded loads SDE data if not already loaded
func (s *Service) ensureLoaded() error {
	// Fast path: data already loaded, no locking needed
	if s.loaded {
		return nil
	}

	s.loadMu.Lock()
	defer s.loadMu.Unlock()

	// Double-check after acquiring lock (another goroutine might have loaded it)
	if s.loaded {
		return nil
	}

	if err := s.loadAgents(); err != nil {
		return fmt.Errorf("failed to load agents: %w", err)
	}

	if err := s.loadCategories(); err != nil {
		return fmt.Errorf("failed to load categories: %w", err)
	}

	if err := s.loadBlueprints(); err != nil {
		return fmt.Errorf("failed to load blueprints: %w", err)
	}

	s.loaded = true
	
	// Log memory usage after loading SDE data
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	slog.Info("SDE data loaded successfully",
		"agents_count", len(s.agents),
		"categories_count", len(s.categories),
		"blueprints_count", len(s.blueprints),
		"heap_size", formatBytes(m.HeapAlloc),
		"total_alloc", formatBytes(m.TotalAlloc),
	)
	
	return nil
}

// loadAgents loads agent data from JSON file
func (s *Service) loadAgents() error {
	filePath := filepath.Join(s.dataDir, "agents.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read agents file: %w", err)
	}

	var agents map[string]*Agent
	if err := json.Unmarshal(data, &agents); err != nil {
		return fmt.Errorf("failed to unmarshal agents: %w", err)
	}

	s.agents = agents
	return nil
}

// loadCategories loads category data from JSON file
func (s *Service) loadCategories() error {
	filePath := filepath.Join(s.dataDir, "categories.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read categories file: %w", err)
	}

	var categories map[string]*Category
	if err := json.Unmarshal(data, &categories); err != nil {
		return fmt.Errorf("failed to unmarshal categories: %w", err)
	}

	s.categories = categories
	return nil
}

// loadBlueprints loads blueprint data from JSON file
func (s *Service) loadBlueprints() error {
	filePath := filepath.Join(s.dataDir, "blueprints.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read blueprints file: %w", err)
	}

	var blueprints map[string]*Blueprint
	if err := json.Unmarshal(data, &blueprints); err != nil {
		return fmt.Errorf("failed to unmarshal blueprints: %w", err)
	}

	s.blueprints = blueprints
	return nil
}

// GetAgent retrieves an agent by ID
func (s *Service) GetAgent(id string) (*Agent, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	agent, exists := s.agents[id]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", id)
	}

	return agent, nil
}

// GetCategory retrieves a category by ID
func (s *Service) GetCategory(id string) (*Category, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	category, exists := s.categories[id]
	if !exists {
		return nil, fmt.Errorf("category %s not found", id)
	}

	return category, nil
}

// GetBlueprint retrieves a blueprint by ID
func (s *Service) GetBlueprint(id string) (*Blueprint, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	blueprint, exists := s.blueprints[id]
	if !exists {
		return nil, fmt.Errorf("blueprint %s not found", id)
	}

	return blueprint, nil
}

// GetAgentsByLocation returns all agents at a specific location
func (s *Service) GetAgentsByLocation(locationID int) ([]*Agent, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	var agents []*Agent
	for _, agent := range s.agents {
		if agent.LocationID == locationID {
			agents = append(agents, agent)
		}
	}

	return agents, nil
}

// GetPublishedCategories returns all published categories
func (s *Service) GetPublishedCategories() (map[string]*Category, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	published := make(map[string]*Category)
	for id, category := range s.categories {
		if category.Published {
			published[id] = category
		}
	}

	return published, nil
}

// GetAllAgents returns all agents
func (s *Service) GetAllAgents() (map[string]*Agent, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	// Return a copy to prevent external modification
	agents := make(map[string]*Agent, len(s.agents))
	for id, agent := range s.agents {
		agents[id] = agent
	}

	return agents, nil
}

// GetAllCategories returns all categories
func (s *Service) GetAllCategories() (map[string]*Category, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	// Return a copy to prevent external modification
	categories := make(map[string]*Category, len(s.categories))
	for id, category := range s.categories {
		categories[id] = category
	}

	return categories, nil
}

// GetAllBlueprints returns all blueprints
func (s *Service) GetAllBlueprints() (map[string]*Blueprint, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	// Return a copy to prevent external modification
	blueprints := make(map[string]*Blueprint, len(s.blueprints))
	for id, blueprint := range s.blueprints {
		blueprints[id] = blueprint
	}

	return blueprints, nil
}

// IsLoaded returns whether SDE data has been loaded
func (s *Service) IsLoaded() bool {
	return s.loaded
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}