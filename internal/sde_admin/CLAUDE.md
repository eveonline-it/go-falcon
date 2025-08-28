# SDE Admin Module (internal/sde_admin)

## Overview

The SDE Admin module provides administrative functionality for managing EVE Online Static Data Export (SDE) data in memory. It allows super administrators to monitor, reload, and verify SDE data that is loaded from the local file system into application memory for ultra-high-performance access.

## Architecture

### Core Components

- **Memory Monitoring**: Real-time visibility into in-memory SDE data status and usage, including universe data
- **Data Reloading**: Hot reload SDE data from files without application restart
- **Integrity Verification**: Validate completeness and consistency of loaded data across all 46 data types
- **Performance Statistics**: Monitor memory usage and item counts per data type (including 9,725+ universe files)
- **System Information**: Runtime metrics and memory utilization tracking
- **Universe Management**: Complete EVE universe administration (regions, constellations, solar systems)

### Files Structure

```
internal/sde_admin/
├── dto/                 # Data transfer objects for API requests/responses
│   ├── inputs.go       # Reload request structures
│   └── outputs.go      # Response structures and converters
├── routes/             # HTTP route handlers
│   └── routes.go       # Huma v2 route registrations for memory management
├── services/           # Business logic
│   └── service.go      # SDE in-memory data inspection and management
├── module.go           # Module initialization and integration
└── CLAUDE.md           # This documentation
```

## SDE In-Memory Data Management

### Fully Supported Data Types (46 total)

The module can monitor and manage all SDE data types supported by the `pkg/sde` service - complete coverage of all EVE Online SDE data files loaded in memory, including universe data:

- **agents**: Mission agents with location and corporation info
- **categories**: Item categories with internationalized names
- **blueprints**: Manufacturing blueprints with material requirements
- **marketGroups**: Market categorization and hierarchy
- **metaGroups**: Item meta group classifications
- **npcCorporations**: NPC corporation data with faction info
- **typeIDs**: Basic type information (lightweight)
- **types**: Complete item type database with attributes
- **typeMaterials**: Manufacturing material requirements per type
- **races**: Character races with skills and ship information
- **factions**: EVE factions with member races and corporation relationships
- **bloodlines**: Character bloodlines with attributes and racial information
- **groups**: Item groups with categorization and property flags
- **dogmaAttributes**: Item attributes and properties for EVE mechanics
- **ancestries**: Character ancestry information with bloodline relationships
- **certificates**: Skill certificates with tiered requirements (basic → elite)
- **characterAttributes**: Core character attributes (Intelligence, Charisma, etc.)
- **skins**: Ship skins with visibility and material information
- **staStations**: Station data with location and service information
- **dogmaEffects**: Dogma effects with modifiers and attributes
- **iconIDs**: Icon ID to file path mappings
- **graphicIDs**: Graphic ID definitions with SOF data
- **typeDogma**: Type dogma attributes and effects for items
- **invFlags**: Inventory flag definitions with names and order
- **stationServices**: Station service definitions with internationalized names
- **stationOperations**: Station operations with manufacturing factors
- **researchAgents**: Research agents with skill requirements
- **agentsInSpace**: Agents located in space with dungeon and location information
- **contrabandTypes**: Contraband types with faction-specific penalties and restrictions
- **corporationActivities**: Corporation activities with internationalized names
- **invItems**: Inventory items with location, ownership, and quantity information
- **npcCorporationDivisions**: NPC corporation divisions with internationalized names
- **controlTowerResources**: Control tower resource requirements with faction and security restrictions
- **dogmaAttributeCategories**: Dogma attribute categories with names and descriptions
- **invNames**: Inventory names mapping item IDs to readable names
- **invPositions**: Inventory position and orientation data for items in space
- **invUniqueNames**: Unique inventory names with group information
- **planetResources**: Planet resource power and workforce requirements
- **planetSchematics**: Planetary interaction schematics with cycle times and material flows
- **skinLicenses**: Ship skin licenses with duration and type information
- **skinMaterials**: Skin material definitions with display names and material sets
- **sovereigntyUpgrades**: Sovereignty upgrade specifications with fuel and resource costs
- **translationLanguages**: Language code to name mappings for internationalization

**Universe Data** (3 data types - 9,725 files total):
- **regions**: EVE Online regions with boundaries, factions, and nebula information (113 files)
- **constellations**: Star constellation data with positioning and radius information (1,175 files)  
- **solarSystems**: Complete solar system data including planets, moons, stations, stargates, and asteroid belts (8,437 files)

**Complete Implementation**: All 46 data types from the EVE Online Static Data Export are now fully implemented and available for import, including the complete EVE universe structure.

### In-Memory Storage Structure

SDE data is stored in application memory using optimized Go data structures in the `pkg/sde` service:

- **Maps**: Fast O(1) lookups for keyed data (agents, types, categories, etc.)
- **Slices**: Array-based data for ordered information (stations, inventory items, etc.)
- **Pointers**: Memory-efficient storage with shared references
- **Type Safety**: Strongly typed structs with JSON unmarshaling support

**Memory Benefits**:
- **Ultra-fast access**: Nanosecond lookups vs milliseconds for network calls
- **No serialization overhead**: Direct struct access vs JSON parsing  
- **Efficient memory usage**: ~400MB for all data including complete EVE universe
- **Thread-safe**: Concurrent read access with minimal locking
- **Complete EVE universe**: In-memory access to all 9,725 universe files (regions, constellations, solar systems)

### Data Management Process

1. **Loading**: Automatic lazy-loading of SDE data on first access from JSON files
2. **Monitoring**: Real-time visibility into loaded data types, counts, and memory usage
3. **Reloading**: Hot reload individual data types or complete SDE dataset
4. **Verification**: Integrity checks to ensure data completeness and consistency
5. **Statistics**: Detailed memory usage and performance metrics
6. **System Info**: Runtime monitoring with Go runtime statistics

## API Endpoints

### Public Endpoints

#### Get Module Status
```
GET /sde_admin/status
```
Returns the health and status of the SDE admin module.

**Response:**
```json
{
  "body": {
    "module": "sde_admin",
    "status": "healthy",
    "message": "SDE admin module is operational for in-memory data management"
  }
}
```

### Administrative Endpoints

All administrative endpoints require **Super Administrator** permissions.

#### Get SDE Memory Status
```
GET /sde_admin/memory
```
**Authentication:** Super Admin Required
**Permission:** Super Administrator group membership

Get detailed status of SDE data currently loaded in memory.

**Response:**
```json
{
  "body": {
    "loaded_data_types": ["agents", "types", "categories"],
    "total_data_types": 46,
    "memory_usage": {
      "total_estimated_mb": 245.7,
      "data_types": {
        "agents": {
          "count": 15420,
          "estimated_memory_mb": 12.3,
          "status": "loaded"
        },
        "types": {
          "count": 87451,
          "estimated_memory_mb": 156.8,
          "status": "loaded"
        }
      }
    },
    "last_reload": "2024-01-15T10:35:22Z"
  }
}
```

#### Get SDE Statistics
```
GET /sde_admin/stats
```
**Authentication:** Super Admin Required

Get detailed statistics about SDE data loaded in memory including performance metrics.

**Response:**
```json
{
  "body": {
    "total_data_types": 46,
    "loaded_data_types": 35,
    "total_items": 125843,
    "memory_usage": {
      "estimated_total_mb": 245.7,
      "go_runtime_mb": 312.5,
      "system_memory_mb": 1024.0
    },
    "data_types": {
      "agents": {
        "count": 15420,
        "estimated_memory_mb": 12.3,
        "last_accessed": "2024-01-15T10:30:00Z"
      },
      "types": {
        "count": 87451,
        "estimated_memory_mb": 156.8,
        "last_accessed": "2024-01-15T10:32:15Z"
      }
    },
    "performance": {
      "average_access_time_ns": 150,
      "total_access_count": 45632
    }
  }
}
```

#### Reload SDE Data
```
POST /sde_admin/reload
```
**Authentication:** Super Admin Required

Reload SDE data from files into memory. Can reload all data types or specific ones.

**Request Body:**
```json
{
  "data_types": ["agents", "types", "categories"]  // Optional: specific data types
}
```

**Response:**
```json
{
  "body": {
    "success": true,
    "message": "Successfully reloaded 3 data types from files",
    "reloaded_data_types": ["agents", "types", "categories"],
    "total_items": 125843,
    "duration_seconds": 2.45,
    "memory_usage_mb": 245.7
  }
}
```

**Parameters:**
- `data_types`: Array of data type names to reload (empty = all types)

#### Verify SDE Data Integrity
```
GET /sde_admin/verify
```
**Authentication:** Super Admin Required

Verify the integrity and completeness of loaded SDE data.

**Response:**
```json
{
  "body": {
    "valid": true,
    "message": "All SDE data integrity checks passed",
    "checks": {
      "data_completeness": {
        "valid": true,
        "loaded_types": 46,
        "expected_types": 46,
        "missing_types": []
      },
      "data_consistency": {
        "valid": true,
        "total_items": 125843,
        "validation_errors": []
      },
      "memory_integrity": {
        "valid": true,
        "estimated_memory_mb": 245.7,
        "fragmentation_ratio": 0.12
      }
    },
    "verification_time": "2024-01-15T10:35:22Z"
  }
}
```

#### Get System Information
```
GET /sde_admin/system
```
**Authentication:** Super Admin Required

Get system information relevant to SDE data management including memory usage and operational status.

**Response:**
```json
{
  "body": {
    "is_loaded": true,
    "status": "loaded",
    "loaded_data_types": 46,
    "estimated_memory_mb": 245.7,
    "system_memory_mb": 312.5,
    "go_routines": 45,
    "timestamp": "2024-01-15T10:35:22Z"
  }
}
```

**Status Field Values:**
- `loaded` - SDE has been loaded correctly and is ready for use
- `downloading` - Currently downloading new SDE data from CCP
- `extracting` - Extracting SDE zip archive  
- `converting` - Converting YAML files to JSON format
- `loading` - Loading data into memory
- `error` - Error occurred during operations

## Memory Management System

### Data Type Status Model

Memory-loaded data is tracked with comprehensive status information:

```go
type DataTypeStatus struct {
    Name              string    // Data type name (e.g., "agents", "types")
    Status            string    // "loaded", "not_loaded", "loading", "error"
    Count             int       // Total items loaded for this type
    EstimatedMemoryMB float64   // Estimated memory usage in megabytes
    LastAccessed      time.Time // Last time this data type was accessed
    LastReloaded      time.Time // Last time this data type was reloaded
    Error             string    // Error message if loading failed
}

type DataTypeStats struct {
    Name              string    // Data type name
    Count             int       // Total items in memory
    EstimatedMemoryMB float64   // Memory estimation using reflection
    Status            string    // Current status
}

type LoadStatus struct {
    LoadedDataTypes   []string            // List of currently loaded data types
    TotalDataTypes    int                // Total available data types
    MemoryUsage       MemoryUsageInfo    // Memory usage statistics
    LastReload        time.Time          // Last full or partial reload time
}
```

### Memory Management

- **Lazy Loading**: Data types loaded on first access from JSON files
- **Hot Reload**: Individual data types or complete dataset can be reloaded without restart
- **Thread Safety**: All operations are thread-safe for concurrent access
- **Memory Estimation**: Accurate memory usage calculation using Go reflection

### Real-time Monitoring

Memory status is available in real-time:

1. **Data Type Counts**: Number of items loaded per data type
2. **Memory Usage**: Estimated memory consumption per type and total
3. **Access Tracking**: Last access time for performance monitoring
4. **Reload History**: Track when data was last refreshed from files

## Error Handling

### Memory Management Error Handling

- **Data Type Errors**: Individual data type loading failures are isolated and reported
- **File Access**: Graceful handling of missing or corrupted SDE data files
- **Memory Limits**: Monitoring and warnings for excessive memory usage
- **Reload Failures**: Partial reload failures preserve existing data in memory
- **Error Logging**: Comprehensive error logging with context information

### HTTP Error Responses

- **400 Bad Request**: Invalid request parameters (e.g., invalid data type names)
- **401 Unauthorized**: Missing or invalid authentication token
- **403 Forbidden**: User does not have Super Administrator permissions
- **404 Not Found**: Requested data type or resource not found
- **500 Internal Server Error**: System errors (file access, memory allocation, etc.)

### Error Response Format

```json
{
  "error": {
    "type": "https://example.com/errors/forbidden",
    "title": "Forbidden",
    "status": 403,
    "detail": "Super Administrator permissions required for this operation"
  }
}
```

## Performance Considerations

### Memory Management

- **Lazy Loading**: Data loaded only when accessed, minimizing initial memory footprint
- **Efficient Storage**: Direct Go struct access without serialization overhead
- **Thread Safety**: Concurrent read access with minimal locking overhead
- **Memory Estimation**: Real-time memory usage tracking using reflection

### In-Memory Performance

- **Access Speed**: Nanosecond O(1) lookups vs milliseconds for Redis network calls
- **Memory Usage**: ~300MB for complete SDE dataset (vs ~610MB Redis overhead)
- **No Serialization**: Direct struct access eliminates JSON parsing overhead
- **Cache Efficiency**: CPU cache-friendly data structures improve performance

### Monitoring

- **Real-time Memory Tracking**: Live monitoring of memory usage per data type
- **Access Pattern Analysis**: Track data type usage for optimization
- **System Resource Monitoring**: Go runtime statistics and system memory usage
- **Performance Metrics**: Access time measurements and throughput analysis

## Security Features

### Super Administrator Only

All administrative operations require Super Administrator group membership:
- Memory status operations cannot be accessed by regular users
- Data reloading restricted to administrators
- System information requires highest privilege level
- Statistical data access limited to administrators

### Audit Trail

- **Operation Logging**: All administrative actions logged with operator information
- **Reload History**: Complete history of data reload operations maintained
- **Error Logging**: Failed operations logged with detailed error information
- **Performance Metrics**: Memory usage and access patterns logged for analysis

## Integration Points

### SDE Service Integration

- **Direct Interface**: Direct integration with `pkg/sde` service interface
- **Service Dependency**: SDE admin module depends on active SDE service instance
- **Type Compatibility**: Supports all SDE data types available from the service
- **Error Propagation**: SDE service errors handled gracefully with detailed reporting

### Memory Management Integration

- **Real-time Monitoring**: Direct access to in-memory data structures
- **Hot Reload**: Refresh data from files without application restart
- **Thread Safety**: Safe concurrent access to memory statistics
- **Performance Tracking**: Monitor access patterns and memory usage

### Authentication Integration

- **Centralized Middleware**: Uses `pkg/middleware` for authentication and authorization
- **Super Admin Enforcement**: All operations require super administrator permissions
- **JWT Validation**: Token-based authentication with session management
- **Permission Validation**: Fine-grained access control per endpoint

## Future Enhancements

### Planned Features

- **Incremental Updates**: Update only changed SDE data files
- **Scheduled Reloads**: Automatic data reloads when SDE files are updated
- **Advanced Validation**: Deep data integrity checks with cross-references
- **Export Functionality**: Export in-memory data to various file formats

### Advanced Features

- **Memory Optimization**: Advanced memory pooling and garbage collection tuning
- **Data Compression**: Compress in-memory structures to reduce footprint
- **Reload Templates**: Pre-configured data type reload profiles
- **Notification System**: Alert administrators on reload completion/failure
- **Performance Analytics**: Detailed access pattern analysis and optimization recommendations

## Dependencies

### Internal Dependencies

- `go-falcon/pkg/sde` (SDE service interface for in-memory data access)
- `go-falcon/pkg/middleware` (Centralized authentication and permissions)
- `go-falcon/pkg/handlers` (Standard response utilities)
- `go-falcon/internal/auth/models` (Authenticated user models)

### External Dependencies

- `github.com/danielgtaylor/huma/v2` (API framework with type-safe operations)
- `runtime` (Go runtime statistics for system information)
- `reflect` (Memory estimation and data structure analysis)

This SDE Admin module provides a comprehensive solution for managing EVE Online static data in memory with full administrative control, real-time monitoring, and robust error handling suitable for production environments.