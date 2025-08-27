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
	case models.DataTypeRaces:
		return s.importRaces(ctx, importStatus, batchSize, force)
	case models.DataTypeFactions:
		return s.importFactions(ctx, importStatus, batchSize, force)
	case models.DataTypeBloodlines:
		return s.importBloodlines(ctx, importStatus, batchSize, force)
	case models.DataTypeGroups:
		return s.importGroups(ctx, importStatus, batchSize, force)
	case models.DataTypeDogmaAttributes:
		return s.importDogmaAttributes(ctx, importStatus, batchSize, force)
	case models.DataTypeAncestries:
		return s.importAncestries(ctx, importStatus, batchSize, force)
	case models.DataTypeCertificates:
		return s.importCertificates(ctx, importStatus, batchSize, force)
	case models.DataTypeCharacterAttributes:
		return s.importCharacterAttributes(ctx, importStatus, batchSize, force)
	case models.DataTypeSkins:
		return s.importSkins(ctx, importStatus, batchSize, force)
	case models.DataTypeStaStations:
		return s.importStaStations(ctx, importStatus, batchSize, force)
	case models.DataTypeDogmaEffects:
		return s.importDogmaEffects(ctx, importStatus, batchSize, force)
	case models.DataTypeIconIDs:
		return s.importIconIDs(ctx, importStatus, batchSize, force)
	case models.DataTypeGraphicIDs:
		return s.importGraphicIDs(ctx, importStatus, batchSize, force)
	case models.DataTypeTypeDogma:
		return s.importTypeDogma(ctx, importStatus, batchSize, force)
	case models.DataTypeInvFlags:
		return s.importInvFlags(ctx, importStatus, batchSize, force)
	case models.DataTypeStationServices:
		return s.importStationServices(ctx, importStatus, batchSize, force)
	case models.DataTypeStationOperations:
		return s.importStationOperations(ctx, importStatus, batchSize, force)
	case models.DataTypeResearchAgents:
		return s.importResearchAgents(ctx, importStatus, batchSize, force)
	case models.DataTypeAgentsInSpace:
		return s.importAgentsInSpace(ctx, importStatus, batchSize, force)
	case models.DataTypeContrabandTypes:
		return s.importContrabandTypes(ctx, importStatus, batchSize, force)
	case models.DataTypeCorporationActivities:
		return s.importCorporationActivities(ctx, importStatus, batchSize, force)
	case models.DataTypeInvItems:
		return s.importInvItems(ctx, importStatus, batchSize, force)
	case models.DataTypeNPCCorporationDivisions:
		return s.importNPCCorporationDivisions(ctx, importStatus, batchSize, force)
	case models.DataTypeControlTowerResources:
		return s.importControlTowerResources(ctx, importStatus, batchSize, force)
	case models.DataTypeDogmaAttributeCategories:
		return s.importDogmaAttributeCategories(ctx, importStatus, batchSize, force)
	case models.DataTypeInvNames:
		return s.importInvNames(ctx, importStatus, batchSize, force)
	case models.DataTypeInvPositions:
		return s.importInvPositions(ctx, importStatus, batchSize, force)
	case models.DataTypeInvUniqueNames:
		return s.importInvUniqueNames(ctx, importStatus, batchSize, force)
	case models.DataTypePlanetResources:
		return s.importPlanetResources(ctx, importStatus, batchSize, force)
	case models.DataTypePlanetSchematics:
		return s.importPlanetSchematics(ctx, importStatus, batchSize, force)
	case models.DataTypeSkinLicenses:
		return s.importSkinLicenses(ctx, importStatus, batchSize, force)
	case models.DataTypeSkinMaterials:
		return s.importSkinMaterials(ctx, importStatus, batchSize, force)
	case models.DataTypeSovereigntyUpgrades:
		return s.importSovereigntyUpgrades(ctx, importStatus, batchSize, force)
	case models.DataTypeTranslationLanguages:
		return s.importTranslationLanguages(ctx, importStatus, batchSize, force)
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

// importRaces imports race data to Redis
func (s *Service) importRaces(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	races, err := s.sdeService.GetAllRaces()
	if err != nil {
		return fmt.Errorf("failed to get races: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(races))
	for id, race := range races {
		data[id] = race
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeRaces), data, batchSize, force)
}

// importFactions imports faction data to Redis
func (s *Service) importFactions(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	factions, err := s.sdeService.GetAllFactions()
	if err != nil {
		return fmt.Errorf("failed to get factions: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(factions))
	for id, faction := range factions {
		data[id] = faction
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeFactions), data, batchSize, force)
}

// importBloodlines imports bloodline data to Redis
func (s *Service) importBloodlines(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	bloodlines, err := s.sdeService.GetAllBloodlines()
	if err != nil {
		return fmt.Errorf("failed to get bloodlines: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(bloodlines))
	for id, bloodline := range bloodlines {
		data[id] = bloodline
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeBloodlines), data, batchSize, force)
}

// importGroups imports group data to Redis
func (s *Service) importGroups(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	groups, err := s.sdeService.GetAllGroups()
	if err != nil {
		return fmt.Errorf("failed to get groups: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(groups))
	for id, group := range groups {
		data[id] = group
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeGroups), data, batchSize, force)
}

// importDogmaAttributes imports dogma attribute data to Redis
func (s *Service) importDogmaAttributes(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	dogmaAttributes, err := s.sdeService.GetAllDogmaAttributes()
	if err != nil {
		return fmt.Errorf("failed to get dogma attributes: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(dogmaAttributes))
	for id, dogmaAttribute := range dogmaAttributes {
		data[id] = dogmaAttribute
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeDogmaAttributes), data, batchSize, force)
}

// importAncestries imports ancestry data to Redis
func (s *Service) importAncestries(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	ancestries, err := s.sdeService.GetAllAncestries()
	if err != nil {
		return fmt.Errorf("failed to get ancestries: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(ancestries))
	for id, ancestry := range ancestries {
		data[id] = ancestry
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeAncestries), data, batchSize, force)
}

// importCertificates imports certificate data to Redis
func (s *Service) importCertificates(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	certificates, err := s.sdeService.GetAllCertificates()
	if err != nil {
		return fmt.Errorf("failed to get certificates: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(certificates))
	for id, certificate := range certificates {
		data[id] = certificate
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeCertificates), data, batchSize, force)
}

// importCharacterAttributes imports character attribute data to Redis
func (s *Service) importCharacterAttributes(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	characterAttributes, err := s.sdeService.GetAllCharacterAttributes()
	if err != nil {
		return fmt.Errorf("failed to get character attributes: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(characterAttributes))
	for id, characterAttribute := range characterAttributes {
		data[id] = characterAttribute
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeCharacterAttributes), data, batchSize, force)
}

// importSkins imports skin data to Redis
func (s *Service) importSkins(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	skins, err := s.sdeService.GetAllSkins()
	if err != nil {
		return fmt.Errorf("failed to get skins: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(skins))
	for id, skin := range skins {
		data[id] = skin
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeSkins), data, batchSize, force)
}

// importStaStations imports station data to Redis
func (s *Service) importStaStations(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	stations, err := s.sdeService.GetAllStaStations()
	if err != nil {
		return fmt.Errorf("failed to get stations: %w", err)
	}

	// Update count
	dtStatus := importStatus.Progress.DataTypes[string(models.DataTypeStaStations)]
	dtStatus.Count = len(stations)
	importStatus.Progress.DataTypes[string(models.DataTypeStaStations)] = dtStatus

	// Import stations (stored differently as they're in array format)
	batch := make(map[string]interface{})
	processed := 0

	for _, station := range stations {
		// Use station ID as key
		key := fmt.Sprintf("sde:staStations:%d", station.StationID)
		if !force {
			exists, _ := s.redis.Client.Exists(ctx, key).Result()
			if exists > 0 {
				processed++
				continue
			}
		}

		stationJSON, _ := json.Marshal(station)
		batch[key] = string(stationJSON)

		if len(batch) >= batchSize {
			if err := s.processBatch(ctx, batch); err != nil {
				return err
			}
			processed += len(batch)
			batch = make(map[string]interface{})

			dtStatus.Processed = processed
			importStatus.Progress.DataTypes[string(models.DataTypeStaStations)] = dtStatus
			s.updateImportStatus(ctx, importStatus)
		}
	}

	if len(batch) > 0 {
		if err := s.processBatch(ctx, batch); err != nil {
			return err
		}
		processed += len(batch)
	}

	dtStatus.Processed = len(stations)
	importStatus.Progress.DataTypes[string(models.DataTypeStaStations)] = dtStatus
	s.updateImportStatus(ctx, importStatus)

	return nil
}

// importDogmaEffects imports dogma effect data to Redis
func (s *Service) importDogmaEffects(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	dogmaEffects, err := s.sdeService.GetAllDogmaEffects()
	if err != nil {
		return fmt.Errorf("failed to get dogma effects: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(dogmaEffects))
	for id, effect := range dogmaEffects {
		data[id] = effect
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeDogmaEffects), data, batchSize, force)
}

// importIconIDs imports icon ID data to Redis
func (s *Service) importIconIDs(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	iconIDs, err := s.sdeService.GetAllIconIDs()
	if err != nil {
		return fmt.Errorf("failed to get icon IDs: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(iconIDs))
	for id, iconID := range iconIDs {
		data[id] = iconID
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeIconIDs), data, batchSize, force)
}

// importGraphicIDs imports graphic ID data to Redis
func (s *Service) importGraphicIDs(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	graphicIDs, err := s.sdeService.GetAllGraphicIDs()
	if err != nil {
		return fmt.Errorf("failed to get graphic IDs: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(graphicIDs))
	for id, graphicID := range graphicIDs {
		data[id] = graphicID
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeGraphicIDs), data, batchSize, force)
}

// importTypeDogma imports type dogma data to Redis
func (s *Service) importTypeDogma(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	typeDogma, err := s.sdeService.GetAllTypeDogma()
	if err != nil {
		return fmt.Errorf("failed to get type dogma: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(typeDogma))
	for id, dogma := range typeDogma {
		data[id] = dogma
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeTypeDogma), data, batchSize, force)
}

// importInvFlags imports inventory flags data to Redis
func (s *Service) importInvFlags(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	invFlags, err := s.sdeService.GetAllInvFlags()
	if err != nil {
		return fmt.Errorf("failed to get inventory flags: %w", err)
	}

	// Update count
	dtStatus := importStatus.Progress.DataTypes[string(models.DataTypeInvFlags)]
	dtStatus.Count = len(invFlags)
	importStatus.Progress.DataTypes[string(models.DataTypeInvFlags)] = dtStatus

	// Import flags (stored differently as they're in array format)
	batch := make(map[string]interface{})
	processed := 0

	for _, flag := range invFlags {
		// Use flag ID as key
		key := fmt.Sprintf("sde:invFlags:%d", flag.FlagID)
		if !force {
			exists, _ := s.redis.Client.Exists(ctx, key).Result()
			if exists > 0 {
				processed++
				continue
			}
		}

		flagJSON, _ := json.Marshal(flag)
		batch[key] = string(flagJSON)

		if len(batch) >= batchSize {
			if err := s.processBatch(ctx, batch); err != nil {
				return err
			}
			processed += len(batch)
			batch = make(map[string]interface{})

			dtStatus.Processed = processed
			importStatus.Progress.DataTypes[string(models.DataTypeInvFlags)] = dtStatus
			s.updateImportStatus(ctx, importStatus)
		}
	}

	if len(batch) > 0 {
		if err := s.processBatch(ctx, batch); err != nil {
			return err
		}
		processed += len(batch)
	}

	dtStatus.Processed = len(invFlags)
	importStatus.Progress.DataTypes[string(models.DataTypeInvFlags)] = dtStatus
	s.updateImportStatus(ctx, importStatus)

	return nil
}

// importStationServices imports station services data to Redis
func (s *Service) importStationServices(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	stationServices, err := s.sdeService.GetAllStationServices()
	if err != nil {
		return fmt.Errorf("failed to get station services: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(stationServices))
	for id, service := range stationServices {
		data[id] = service
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeStationServices), data, batchSize, force)
}

// importStationOperations imports station operations data to Redis
func (s *Service) importStationOperations(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	stationOperations, err := s.sdeService.GetAllStationOperations()
	if err != nil {
		return fmt.Errorf("failed to get station operations: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(stationOperations))
	for id, operation := range stationOperations {
		data[id] = operation
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeStationOperations), data, batchSize, force)
}

// importResearchAgents imports research agents data to Redis
func (s *Service) importResearchAgents(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	researchAgents, err := s.sdeService.GetAllResearchAgents()
	if err != nil {
		return fmt.Errorf("failed to get research agents: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(researchAgents))
	for id, agent := range researchAgents {
		data[id] = agent
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeResearchAgents), data, batchSize, force)
}

// importAgentsInSpace imports agents in space data to Redis
func (s *Service) importAgentsInSpace(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	agentsInSpace, err := s.sdeService.GetAllAgentsInSpace()
	if err != nil {
		return fmt.Errorf("failed to get agents in space: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(agentsInSpace))
	for id, agent := range agentsInSpace {
		data[id] = agent
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeAgentsInSpace), data, batchSize, force)
}

// importContrabandTypes imports contraband types data to Redis
func (s *Service) importContrabandTypes(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	contrabandTypes, err := s.sdeService.GetAllContrabandTypes()
	if err != nil {
		return fmt.Errorf("failed to get contraband types: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(contrabandTypes))
	for id, contraband := range contrabandTypes {
		data[id] = contraband
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeContrabandTypes), data, batchSize, force)
}

// importCorporationActivities imports corporation activities data to Redis
func (s *Service) importCorporationActivities(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	corporationActivities, err := s.sdeService.GetAllCorporationActivities()
	if err != nil {
		return fmt.Errorf("failed to get corporation activities: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(corporationActivities))
	for id, activity := range corporationActivities {
		data[id] = activity
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeCorporationActivities), data, batchSize, force)
}

// importInvItems imports inventory items data to Redis
func (s *Service) importInvItems(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	invItems, err := s.sdeService.GetAllInvItems()
	if err != nil {
		return fmt.Errorf("failed to get inventory items: %w", err)
	}

	// Convert to map[string]interface{} using itemID as key
	data := make(map[string]interface{}, len(invItems))
	for _, item := range invItems {
		data[fmt.Sprintf("%d", item.ItemID)] = item
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeInvItems), data, batchSize, force)
}

// importNPCCorporationDivisions imports NPC corporation divisions data to Redis
func (s *Service) importNPCCorporationDivisions(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	npcCorporationDivisions, err := s.sdeService.GetAllNPCCorporationDivisions()
	if err != nil {
		return fmt.Errorf("failed to get NPC corporation divisions: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(npcCorporationDivisions))
	for id, division := range npcCorporationDivisions {
		data[id] = division
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeNPCCorporationDivisions), data, batchSize, force)
}

// importControlTowerResources imports control tower resources data to Redis
func (s *Service) importControlTowerResources(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	controlTowerResources, err := s.sdeService.GetAllControlTowerResources()
	if err != nil {
		return fmt.Errorf("failed to get control tower resources: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(controlTowerResources))
	for id, resources := range controlTowerResources {
		data[id] = resources
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeControlTowerResources), data, batchSize, force)
}

// importDogmaAttributeCategories imports dogma attribute categories data to Redis
func (s *Service) importDogmaAttributeCategories(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	dogmaAttributeCategories, err := s.sdeService.GetAllDogmaAttributeCategories()
	if err != nil {
		return fmt.Errorf("failed to get dogma attribute categories: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(dogmaAttributeCategories))
	for id, category := range dogmaAttributeCategories {
		data[id] = category
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeDogmaAttributeCategories), data, batchSize, force)
}

// importInvNames imports inventory names data to Redis
func (s *Service) importInvNames(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	invNames, err := s.sdeService.GetAllInvNames()
	if err != nil {
		return fmt.Errorf("failed to get inventory names: %w", err)
	}

	// Convert to map[string]interface{} using itemID as key
	data := make(map[string]interface{}, len(invNames))
	for _, name := range invNames {
		data[fmt.Sprintf("%d", name.ItemID)] = name
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeInvNames), data, batchSize, force)
}

// importInvPositions imports inventory positions data to Redis
func (s *Service) importInvPositions(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	invPositions, err := s.sdeService.GetAllInvPositions()
	if err != nil {
		return fmt.Errorf("failed to get inventory positions: %w", err)
	}

	// Convert to map[string]interface{} using itemID as key
	data := make(map[string]interface{}, len(invPositions))
	for _, position := range invPositions {
		data[fmt.Sprintf("%d", position.ItemID)] = position
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeInvPositions), data, batchSize, force)
}

// importInvUniqueNames imports inventory unique names data to Redis
func (s *Service) importInvUniqueNames(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	invUniqueNames, err := s.sdeService.GetAllInvUniqueNames()
	if err != nil {
		return fmt.Errorf("failed to get inventory unique names: %w", err)
	}

	// Convert to map[string]interface{} using itemID as key
	data := make(map[string]interface{}, len(invUniqueNames))
	for _, uniqueName := range invUniqueNames {
		data[fmt.Sprintf("%d", uniqueName.ItemID)] = uniqueName
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeInvUniqueNames), data, batchSize, force)
}

// importPlanetResources imports planet resources data to Redis
func (s *Service) importPlanetResources(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	planetResources, err := s.sdeService.GetAllPlanetResources()
	if err != nil {
		return fmt.Errorf("failed to get planet resources: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(planetResources))
	for id, resource := range planetResources {
		data[id] = resource
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypePlanetResources), data, batchSize, force)
}

// importPlanetSchematics imports planet schematics data to Redis
func (s *Service) importPlanetSchematics(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	planetSchematics, err := s.sdeService.GetAllPlanetSchematics()
	if err != nil {
		return fmt.Errorf("failed to get planet schematics: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(planetSchematics))
	for id, schematic := range planetSchematics {
		data[id] = schematic
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypePlanetSchematics), data, batchSize, force)
}

// importSkinLicenses imports skin licenses data to Redis
func (s *Service) importSkinLicenses(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	skinLicenses, err := s.sdeService.GetAllSkinLicenses()
	if err != nil {
		return fmt.Errorf("failed to get skin licenses: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(skinLicenses))
	for id, license := range skinLicenses {
		data[id] = license
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeSkinLicenses), data, batchSize, force)
}

// importSkinMaterials imports skin materials data to Redis
func (s *Service) importSkinMaterials(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	skinMaterials, err := s.sdeService.GetAllSkinMaterials()
	if err != nil {
		return fmt.Errorf("failed to get skin materials: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(skinMaterials))
	for id, material := range skinMaterials {
		data[id] = material
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeSkinMaterials), data, batchSize, force)
}

// importSovereigntyUpgrades imports sovereignty upgrades data to Redis
func (s *Service) importSovereigntyUpgrades(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	sovereigntyUpgrades, err := s.sdeService.GetAllSovereigntyUpgrades()
	if err != nil {
		return fmt.Errorf("failed to get sovereignty upgrades: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(sovereigntyUpgrades))
	for id, upgrade := range sovereigntyUpgrades {
		data[id] = upgrade
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeSovereigntyUpgrades), data, batchSize, force)
}

// importTranslationLanguages imports translation languages data to Redis
func (s *Service) importTranslationLanguages(ctx context.Context, importStatus *models.ImportStatus, batchSize int, force bool) error {
	translationLanguages, err := s.sdeService.GetAllTranslationLanguages()
	if err != nil {
		return fmt.Errorf("failed to get translation languages: %w", err)
	}

	// Convert to map[string]interface{} for generic helper
	data := make(map[string]interface{}, len(translationLanguages))
	for code, language := range translationLanguages {
		data[code] = language
	}

	return s.importGenericData(ctx, importStatus, string(models.DataTypeTranslationLanguages), data, batchSize, force)
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
