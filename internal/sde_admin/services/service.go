package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"go-falcon/internal/sde_admin/dto"
	"go-falcon/internal/sde_admin/models"
	"go-falcon/pkg/database"
	"go-falcon/pkg/sde"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Service handles SDE admin operations
type Service struct {
	mongodb    *database.MongoDB
	redis      *database.Redis
	sdeService sde.SDEService

	// Import status tracking
	activeImports sync.Map // map[string]*models.ImportStatus
	mutex         sync.RWMutex
}

// NewService creates a new SDE admin service
func NewService(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService) *Service {
	return &Service{
		mongodb:    mongodb,
		redis:      redis,
		sdeService: sdeService,
	}
}

// StartImport begins an SDE import operation
func (s *Service) StartImport(ctx context.Context, req *dto.ImportSDERequest) (*dto.ImportSDEResponse, error) {
	// Generate unique import ID
	importID := uuid.New().String()

	// Determine which data types to import
	dataTypes := req.DataTypes
	if len(dataTypes) == 0 {
		// Import all data types
		allTypes := models.GetAllDataTypes()
		for _, dt := range allTypes {
			dataTypes = append(dataTypes, string(dt))
		}
	}

	// Validate data types
	validTypes := make(map[string]bool)
	for _, dt := range models.GetAllDataTypes() {
		validTypes[string(dt)] = true
	}

	for _, dt := range dataTypes {
		if !validTypes[dt] {
			return nil, fmt.Errorf("invalid data type: %s", dt)
		}
	}

	// Set default batch size
	batchSize := req.BatchSize
	if batchSize == 0 {
		batchSize = 1000
	}

	// Create import status
	now := time.Now()
	importStatus := &models.ImportStatus{
		ID:        importID,
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
		Progress: models.ImportProgress{
			TotalSteps:     len(dataTypes),
			CompletedSteps: 0,
			CurrentStep:    "Initializing import",
			DataTypes:      make(map[string]models.DataTypeStatus),
		},
	}

	// Initialize data type statuses
	for _, dt := range dataTypes {
		importStatus.Progress.DataTypes[dt] = models.DataTypeStatus{
			Name:      dt,
			Status:    "pending",
			Count:     0,
			Processed: 0,
		}
	}

	// Store in memory for quick access
	s.activeImports.Store(importID, importStatus)

	// Store in MongoDB for persistence
	collection := s.mongodb.Database.Collection("sde_import_status")
	if _, err := collection.InsertOne(ctx, importStatus); err != nil {
		s.activeImports.Delete(importID)
		return nil, fmt.Errorf("failed to store import status: %w", err)
	}

	// Start the import process in background
	go s.runImport(context.Background(), importID, dataTypes, batchSize, req.Force)

	response := &dto.ImportSDEResponse{
		ImportID:  importID,
		Status:    "pending",
		Message:   fmt.Sprintf("Import started for %d data types", len(dataTypes)),
		StartTime: now.Format(time.RFC3339),
	}

	return response, nil
}

// GetImportStatus retrieves the status of an import operation
func (s *Service) GetImportStatus(ctx context.Context, importID string) (*dto.ImportStatusResponse, error) {
	// First try to get from memory (active imports)
	if status, ok := s.activeImports.Load(importID); ok {
		importStatus := status.(*models.ImportStatus)
		return dto.ConvertFromModel(importStatus), nil
	}

	// If not in memory, try MongoDB
	collection := s.mongodb.Database.Collection("sde_import_status")
	var importStatus models.ImportStatus

	filter := bson.M{"_id": importID}
	err := collection.FindOne(ctx, filter).Decode(&importStatus)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("import not found: %s", importID)
		}
		return nil, fmt.Errorf("failed to retrieve import status: %w", err)
	}

	return dto.ConvertFromModel(&importStatus), nil
}

// GetSDEStats returns statistics about SDE data in Redis
func (s *Service) GetSDEStats(ctx context.Context) (*dto.SDEStatsResponse, error) {
	stats := &dto.SDEStatsResponse{
		DataTypes: make(map[string]dto.DataTypeStats),
	}

	// Get total SDE keys
	keys, err := s.redis.Client.Keys(ctx, "sde:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get SDE keys: %w", err)
	}
	stats.TotalKeys = len(keys)

	// Get memory usage
	memInfo, err := s.redis.Client.Info(ctx, "memory").Result()
	if err == nil {
		for _, line := range strings.Split(memInfo, "\r\n") {
			if strings.HasPrefix(line, "used_memory_human:") {
				stats.RedisMemoryUsed = strings.TrimPrefix(line, "used_memory_human:")
				break
			}
		}
	}

	// Count keys for each data type
	for _, dt := range models.GetAllDataTypes() {
		pattern := fmt.Sprintf("sde:%s:*", string(dt))
		typeKeys, err := s.redis.Client.Keys(ctx, pattern).Result()
		if err != nil {
			continue
		}

		stats.DataTypes[string(dt)] = dto.DataTypeStats{
			Count:      len(typeKeys),
			KeyPattern: pattern,
		}
	}

	// Get last import time from Redis metadata
	lastImport, err := s.redis.Client.Get(ctx, "sde:metadata:last_import").Result()
	if err == nil {
		stats.LastImport = &lastImport
	}

	return stats, nil
}

// ClearSDE removes all SDE data from Redis
func (s *Service) ClearSDE(ctx context.Context) (*dto.ClearSDEResponse, error) {
	// Get all SDE keys
	keys, err := s.redis.Client.Keys(ctx, "sde:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get SDE keys: %w", err)
	}

	if len(keys) == 0 {
		return &dto.ClearSDEResponse{
			Success:     true,
			Message:     "No SDE data found in Redis",
			KeysDeleted: 0,
		}, nil
	}

	// Delete all SDE keys
	deleted, err := s.redis.Client.Del(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to delete SDE keys: %w", err)
	}

	return &dto.ClearSDEResponse{
		Success:     true,
		Message:     fmt.Sprintf("Successfully deleted %d SDE keys", int(deleted)),
		KeysDeleted: int(deleted),
	}, nil
}

// runImport executes the actual import process
func (s *Service) runImport(ctx context.Context, importID string, dataTypes []string, batchSize int, force bool) {
	// Get import status
	statusInterface, exists := s.activeImports.Load(importID)
	if !exists {
		slog.Error("Import status not found", "import_id", importID)
		return
	}

	importStatus := statusInterface.(*models.ImportStatus)

	// Update status to running
	now := time.Now()
	importStatus.Status = "running"
	importStatus.StartTime = &now
	importStatus.Progress.CurrentStep = "Starting import process"
	importStatus.UpdatedAt = now
	s.updateImportStatus(ctx, importStatus)

	defer func() {
		// Clean up from active imports when done
		s.activeImports.Delete(importID)
	}()

	// Ensure SDE service is loaded
	if !s.sdeService.IsLoaded() {
		importStatus.Progress.CurrentStep = "Loading SDE service"
		s.updateImportStatus(ctx, importStatus)

		// Try to load any data to trigger SDE loading
		_, err := s.sdeService.GetAllAgents()
		if err != nil {
			s.failImport(ctx, importStatus, fmt.Errorf("failed to load SDE service: %w", err))
			return
		}
	}

	// Import each data type
	for i, dataType := range dataTypes {
		importStatus.Progress.CurrentStep = fmt.Sprintf("Processing %s (%d/%d)", dataType, i+1, len(dataTypes))
		s.updateImportStatus(ctx, importStatus)

		// Update data type status
		dtStatus := importStatus.Progress.DataTypes[dataType]
		dtStatus.Status = "processing"
		importStatus.Progress.DataTypes[dataType] = dtStatus
		s.updateImportStatus(ctx, importStatus)

		// Import the specific data type
		err := s.importDataType(ctx, importStatus, models.SDEDataType(dataType), batchSize, force)
		if err != nil {
			dtStatus.Status = "failed"
			dtStatus.Error = err.Error()
			importStatus.Progress.DataTypes[dataType] = dtStatus
			s.updateImportStatus(ctx, importStatus)

			s.failImport(ctx, importStatus, fmt.Errorf("failed to import %s: %w", dataType, err))
			return
		}

		// Mark data type as completed
		dtStatus.Status = "completed"
		importStatus.Progress.DataTypes[dataType] = dtStatus
		importStatus.Progress.CompletedSteps++
		s.updateImportStatus(ctx, importStatus)
	}

	// Mark import as completed
	endTime := time.Now()
	importStatus.Status = "completed"
	importStatus.EndTime = &endTime
	importStatus.Progress.CurrentStep = "Import completed successfully"
	importStatus.UpdatedAt = endTime
	s.updateImportStatus(ctx, importStatus)

	// Update last import timestamp in Redis
	s.redis.Client.Set(ctx, "sde:metadata:last_import", endTime.Format(time.RFC3339), 0)

	slog.Info("SDE import completed successfully", "import_id", importID, "duration", endTime.Sub(*importStatus.StartTime))
}

// importDataType imports a specific data type to Redis
func (s *Service) importDataType(ctx context.Context, importStatus *models.ImportStatus, dataType models.SDEDataType, batchSize int, force bool) error {
	switch dataType {
	case models.DataTypeAgents:
		return s.importAgents(ctx, importStatus, batchSize, force)
	case models.DataTypeCategories:
		return s.importCategories(ctx, importStatus, batchSize, force)
	case models.DataTypeBlueprints:
		return s.importBlueprints(ctx, importStatus, batchSize, force)
	case models.DataTypeMarketGroups:
		return s.importMarketGroups(ctx, importStatus, batchSize, force)
	case models.DataTypeMetaGroups:
		return s.importMetaGroups(ctx, importStatus, batchSize, force)
	case models.DataTypeNPCCorporations:
		return s.importNPCCorporations(ctx, importStatus, batchSize, force)
	case models.DataTypeTypeIDs:
		return s.importTypeIDs(ctx, importStatus, batchSize, force)
	case models.DataTypeTypes:
		return s.importTypes(ctx, importStatus, batchSize, force)
	case models.DataTypeTypeMaterials:
		return s.importTypeMaterials(ctx, importStatus, batchSize, force)
	default:
		return fmt.Errorf("unknown data type: %s", dataType)
	}
}

// importAgents imports agent data to Redis
func (s *Service) importAgents(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	agents, err := s.sdeService.GetAllAgents()
	if err != nil {
		return fmt.Errorf("failed to get agents: %w", err)
	}

	// Update count
	dtStatus := importStatus.Progress.DataTypes[string(models.DataTypeAgents)]
	dtStatus.Count = len(agents)
	importStatus.Progress.DataTypes[string(models.DataTypeAgents)] = dtStatus

	// Import in batches
	batch := make(map[string]interface{})
	processed := 0

	for id, agent := range agents {
		// Check if key exists and skip if not forcing
		key := fmt.Sprintf("sde:agents:%s", id)
		if !force {
			exists, _ := s.redis.Client.Exists(ctx, key).Result()
			if exists > 0 {
				processed++
				continue
			}
		}

		// Add to batch
		agentJSON, _ := json.Marshal(agent)
		batch[key] = string(agentJSON)

		// Process batch when full
		if len(batch) >= batchSize {
			if err := s.processBatch(ctx, batch); err != nil {
				return err
			}
			processed += len(batch)
			batch = make(map[string]interface{})

			// Update progress
			dtStatus.Processed = processed
			importStatus.Progress.DataTypes[string(models.DataTypeAgents)] = dtStatus
			s.updateImportStatus(ctx, importStatus)
		}
	}

	// Process remaining items
	if len(batch) > 0 {
		if err := s.processBatch(ctx, batch); err != nil {
			return err
		}
		processed += len(batch)
	}

	// Final update
	dtStatus.Processed = len(agents)
	importStatus.Progress.DataTypes[string(models.DataTypeAgents)] = dtStatus
	s.updateImportStatus(ctx, importStatus)

	return nil
}

// Helper method to import other data types (simplified for brevity)
func (s *Service) importCategories(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	categories, err := s.sdeService.GetAllCategories()
	if err != nil {
		return fmt.Errorf("failed to get categories: %w", err)
	}

	// Convert to interface{} map
	genericData := make(map[string]interface{}, len(categories))
	for id, category := range categories {
		genericData[id] = category
	}
	return s.importGenericData(ctx, importStatus, "categories", genericData, batchSize, force)
}

func (s *Service) importBlueprints(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	blueprints, err := s.sdeService.GetAllBlueprints()
	if err != nil {
		return fmt.Errorf("failed to get blueprints: %w", err)
	}

	// Convert to interface{} map
	genericData := make(map[string]interface{}, len(blueprints))
	for id, blueprint := range blueprints {
		genericData[id] = blueprint
	}
	return s.importGenericData(ctx, importStatus, "blueprints", genericData, batchSize, force)
}

func (s *Service) importMarketGroups(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	marketGroups, err := s.sdeService.GetAllMarketGroups()
	if err != nil {
		return fmt.Errorf("failed to get market groups: %w", err)
	}

	// Convert to interface{} map
	genericData := make(map[string]interface{}, len(marketGroups))
	for id, marketGroup := range marketGroups {
		genericData[id] = marketGroup
	}
	return s.importGenericData(ctx, importStatus, "marketGroups", genericData, batchSize, force)
}

func (s *Service) importMetaGroups(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	metaGroups, err := s.sdeService.GetAllMetaGroups()
	if err != nil {
		return fmt.Errorf("failed to get meta groups: %w", err)
	}

	// Convert to interface{} map
	genericData := make(map[string]interface{}, len(metaGroups))
	for id, metaGroup := range metaGroups {
		genericData[id] = metaGroup
	}
	return s.importGenericData(ctx, importStatus, "metaGroups", genericData, batchSize, force)
}

func (s *Service) importNPCCorporations(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	npcCorps, err := s.sdeService.GetAllNPCCorporations()
	if err != nil {
		return fmt.Errorf("failed to get NPC corporations: %w", err)
	}

	// Convert to interface{} map
	genericData := make(map[string]interface{}, len(npcCorps))
	for id, npcCorp := range npcCorps {
		genericData[id] = npcCorp
	}
	return s.importGenericData(ctx, importStatus, "npcCorporations", genericData, batchSize, force)
}

func (s *Service) importTypeIDs(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	typeIDs, err := s.sdeService.GetAllTypeIDs()
	if err != nil {
		return fmt.Errorf("failed to get type IDs: %w", err)
	}

	// Convert to interface{} map
	genericData := make(map[string]interface{}, len(typeIDs))
	for id, typeID := range typeIDs {
		genericData[id] = typeID
	}
	return s.importGenericData(ctx, importStatus, "typeIDs", genericData, batchSize, force)
}

func (s *Service) importTypes(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	types, err := s.sdeService.GetAllTypes()
	if err != nil {
		return fmt.Errorf("failed to get types: %w", err)
	}

	// Convert to interface{} map
	genericData := make(map[string]interface{}, len(types))
	for id, typeData := range types {
		genericData[id] = typeData
	}
	return s.importGenericData(ctx, importStatus, "types", genericData, batchSize, force)
}

func (s *Service) importTypeMaterials(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	// Type materials are structured differently - need special handling
	dtStatus := importStatus.Progress.DataTypes[string(models.DataTypeTypeMaterials)]

	// Get all type IDs first to determine what materials to fetch
	typeIDs, err := s.sdeService.GetAllTypeIDs()
	if err != nil {
		return fmt.Errorf("failed to get type IDs for materials: %w", err)
	}

	dtStatus.Count = len(typeIDs)
	importStatus.Progress.DataTypes[string(models.DataTypeTypeMaterials)] = dtStatus

	batch := make(map[string]interface{})
	processed := 0

	for typeID := range typeIDs {
		materials, err := s.sdeService.GetTypeMaterials(typeID)
		if err != nil {
			// Skip types that don't have materials
			processed++
			continue
		}

		key := fmt.Sprintf("sde:typeMaterials:%s", typeID)
		if !force {
			exists, _ := s.redis.Client.Exists(ctx, key).Result()
			if exists > 0 {
				processed++
				continue
			}
		}

		materialsJSON, _ := json.Marshal(materials)
		batch[key] = string(materialsJSON)

		if len(batch) >= batchSize {
			if err := s.processBatch(ctx, batch); err != nil {
				return err
			}
			processed += len(batch)
			batch = make(map[string]interface{})

			dtStatus.Processed = processed
			importStatus.Progress.DataTypes[string(models.DataTypeTypeMaterials)] = dtStatus
			s.updateImportStatus(ctx, importStatus)
		}
	}

	if len(batch) > 0 {
		if err := s.processBatch(ctx, batch); err != nil {
			return err
		}
		processed += len(batch)
	}

	dtStatus.Processed = processed
	importStatus.Progress.DataTypes[string(models.DataTypeTypeMaterials)] = dtStatus
	s.updateImportStatus(ctx, importStatus)

	return nil
}

// importGenericData is a helper for importing map-based data structures
func (s *Service) importGenericData(ctx context.Context, importStatus *models.ImportStatus, dataTypeName string, data map[string]interface{}, batchSize int, force bool) error {
	dtStatus := importStatus.Progress.DataTypes[dataTypeName]
	dtStatus.Count = len(data)
	importStatus.Progress.DataTypes[dataTypeName] = dtStatus

	batch := make(map[string]interface{})
	processed := 0

	for id, item := range data {
		key := fmt.Sprintf("sde:%s:%s", dataTypeName, id)
		if !force {
			exists, _ := s.redis.Client.Exists(ctx, key).Result()
			if exists > 0 {
				processed++
				continue
			}
		}

		itemJSON, _ := json.Marshal(item)
		batch[key] = string(itemJSON)

		if len(batch) >= batchSize {
			if err := s.processBatch(ctx, batch); err != nil {
				return err
			}
			processed += len(batch)
			batch = make(map[string]interface{})

			dtStatus.Processed = processed
			importStatus.Progress.DataTypes[dataTypeName] = dtStatus
			s.updateImportStatus(ctx, importStatus)
		}
	}

	if len(batch) > 0 {
		if err := s.processBatch(ctx, batch); err != nil {
			return err
		}
		processed += len(batch)
	}

	dtStatus.Processed = len(data)
	importStatus.Progress.DataTypes[dataTypeName] = dtStatus
	s.updateImportStatus(ctx, importStatus)

	return nil
}

// processBatch processes a batch of key-value pairs to Redis
func (s *Service) processBatch(ctx context.Context, batch map[string]interface{}) error {
	if len(batch) == 0 {
		return nil
	}

	pipe := s.redis.Client.Pipeline()
	for key, value := range batch {
		pipe.Set(ctx, key, value, 0) // No expiration
	}

	_, err := pipe.Exec(ctx)
	return err
}

// updateImportStatus updates the import status in both memory and database
func (s *Service) updateImportStatus(ctx context.Context, status *models.ImportStatus) {
	status.UpdatedAt = time.Now()

	// Update in memory
	s.activeImports.Store(status.ID, status)

	// Update in MongoDB
	collection := s.mongodb.Database.Collection("sde_import_status")
	filter := bson.M{"_id": status.ID}
	update := bson.M{"$set": status}

	opts := options.Update().SetUpsert(true)
	if _, err := collection.UpdateOne(ctx, filter, update, opts); err != nil {
		slog.Error("Failed to update import status in database", "import_id", status.ID, "error", err)
	}
}

// failImport marks an import as failed
func (s *Service) failImport(ctx context.Context, status *models.ImportStatus, err error) {
	now := time.Now()
	status.Status = "failed"
	status.EndTime = &now
	status.Error = err.Error()
	status.UpdatedAt = now

	s.updateImportStatus(ctx, status)

	slog.Error("SDE import failed", "import_id", status.ID, "error", err)
}
