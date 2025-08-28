package services

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"go-falcon/internal/sde_admin/dto"
	"go-falcon/pkg/sde"
)

// Service handles SDE admin operations for in-memory data management
type Service struct {
	sdeService sde.SDEService
}

// NewService creates a new SDE admin service
func NewService(sdeService sde.SDEService) *Service {
	return &Service{
		sdeService: sdeService,
	}
}

// GetMemoryStatus returns the current status of in-memory SDE data
func (s *Service) GetMemoryStatus(ctx context.Context) (*dto.MemoryStatusResponse, error) {
	// Get load status from SDE service
	loadStatus := s.sdeService.GetLoadStatus()
	loadedTypes := s.sdeService.GetLoadedDataTypes()
	totalMemory := s.sdeService.GetTotalMemoryUsage()
	isLoaded := s.sdeService.IsLoaded()

	// Convert to response format
	response := dto.ConvertToMemoryStatus(loadStatus, loadedTypes, totalMemory, isLoaded)

	// Populate file paths for each data type
	for name, status := range response.DataTypeStatuses {
		stats := s.sdeService.GetDataTypeStats(name)
		status.FilePath = stats.FilePath
		response.DataTypeStatuses[name] = status
	}

	return response, nil
}

// GetStats returns detailed statistics about in-memory SDE data
func (s *Service) GetStats(ctx context.Context) (*dto.SDEStatsResponse, error) {
	// Get load status from SDE service
	loadStatus := s.sdeService.GetLoadStatus()
	totalMemory := s.sdeService.GetTotalMemoryUsage()
	isLoaded := s.sdeService.IsLoaded()

	// Convert to response format
	response := dto.ConvertToStatsResponse(loadStatus, totalMemory, isLoaded)

	// Populate file paths for each data type
	for name, stats := range response.DataTypes {
		sdeStats := s.sdeService.GetDataTypeStats(name)
		stats.FilePath = sdeStats.FilePath
		response.DataTypes[name] = stats
	}

	return response, nil
}

// ReloadSDE reloads SDE data from files
func (s *Service) ReloadSDE(ctx context.Context, req *dto.ReloadSDERequest) (*dto.ReloadSDEResponse, error) {
	startTime := time.Now()

	// Determine which data types to reload
	dataTypes := req.DataTypes
	var err error

	slog.Info("Starting SDE reload", "data_types", dataTypes)

	if len(dataTypes) == 0 {
		// Reload all data types
		slog.Info("Reloading all SDE data types")
		err = s.sdeService.ReloadAll()
		if err != nil {
			slog.Error("Failed to reload all SDE data", "error", err)
			return &dto.ReloadSDEResponse{
				Success:    false,
				Message:    fmt.Sprintf("Failed to reload SDE data: %v", err),
				Error:      err.Error(),
				ReloadedAt: time.Now().Format(time.RFC3339),
			}, nil
		}

		// Get all loaded data types for response
		dataTypes = s.sdeService.GetLoadedDataTypes()
	} else {
		// Reload specific data types
		slog.Info("Reloading specific SDE data types", "count", len(dataTypes))
		var failedTypes []string

		for _, dataType := range dataTypes {
			if err := s.sdeService.ReloadDataType(dataType); err != nil {
				slog.Error("Failed to reload data type", "data_type", dataType, "error", err)
				failedTypes = append(failedTypes, dataType)
			}
		}

		if len(failedTypes) > 0 {
			return &dto.ReloadSDEResponse{
				Success:    false,
				Message:    fmt.Sprintf("Failed to reload data types: %v", failedTypes),
				Error:      fmt.Sprintf("Failed data types: %v", failedTypes),
				ReloadedAt: time.Now().Format(time.RFC3339),
			}, nil
		}
	}

	duration := time.Since(startTime)

	slog.Info("SDE reload completed", "duration", duration, "data_types_count", len(dataTypes))

	return &dto.ReloadSDEResponse{
		Success:    true,
		Message:    fmt.Sprintf("Successfully reloaded %d data types", len(dataTypes)),
		DataTypes:  dataTypes,
		Duration:   duration.String(),
		ReloadedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// VerifyIntegrity verifies the integrity of loaded SDE data
func (s *Service) VerifyIntegrity(ctx context.Context) (*dto.VerificationResponse, error) {
	loadStatus := s.sdeService.GetLoadStatus()
	loadedTypes := s.sdeService.GetLoadedDataTypes()

	totalTypes := len(loadStatus)
	loadedCount := len(loadedTypes)

	issues := []string{}

	// Check if all data types are loaded
	if loadedCount < totalTypes {
		unloadedCount := totalTypes - loadedCount
		issues = append(issues, fmt.Sprintf("%d data types are not loaded", unloadedCount))
	}

	// Check for data types with zero items (potential loading issues)
	emptyTypes := []string{}
	for name, status := range loadStatus {
		if status.Loaded && status.Count == 0 {
			emptyTypes = append(emptyTypes, name)
		}
	}

	if len(emptyTypes) > 0 {
		issues = append(issues, fmt.Sprintf("Data types with zero items: %v", emptyTypes))
	}

	// Calculate health score
	healthScore := float64(loadedCount) / float64(totalTypes) * 100
	if len(emptyTypes) > 0 {
		healthScore *= 0.9 // Penalize empty data types
	}

	status := "healthy"
	if healthScore < 100 {
		status = "warning"
	}
	if healthScore < 50 {
		status = "critical"
	}

	return &dto.VerificationResponse{
		Status:         status,
		HealthScore:    healthScore,
		TotalDataTypes: totalTypes,
		LoadedTypes:    loadedCount,
		Issues:         issues,
		VerifiedAt:     time.Now().Format(time.RFC3339),
	}, nil
}

// GetSystemInfo returns system information relevant to SDE data management
func (s *Service) GetSystemInfo(ctx context.Context) (*dto.SystemInfoResponse, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	isLoaded := s.sdeService.IsLoaded()
	loadedTypes := len(s.sdeService.GetLoadedDataTypes())
	totalMemory := s.sdeService.GetTotalMemoryUsage()

	return &dto.SystemInfoResponse{
		IsLoaded:          isLoaded,
		LoadedDataTypes:   loadedTypes,
		EstimatedMemoryMB: float64(totalMemory) / 1024 / 1024,
		SystemMemoryMB:    float64(memStats.Alloc) / 1024 / 1024,
		GoRoutines:        runtime.NumGoroutine(),
		Timestamp:         time.Now().Format(time.RFC3339),
	}, nil
}
