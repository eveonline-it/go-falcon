# SDE Admin Module (internal/sde_admin)

## Overview

The SDE Admin module provides administrative functionality for managing EVE Online Static Data Export (SDE) data in Redis. It allows super administrators to import SDE data from the local file system into Redis for high-performance access and provides progress tracking and statistics.

## Architecture

### Core Components

- **Import Management**: Start and monitor SDE data imports from JSON files to Redis
- **Progress Tracking**: Real-time import progress with detailed statistics per data type
- **Data Statistics**: Monitor Redis storage usage and key counts for SDE data
- **Data Management**: Clear SDE data from Redis when needed

### Files Structure

```
internal/sde_admin/
├── dto/                 # Data transfer objects for API requests/responses
│   ├── inputs.go       # Import request structures
│   └── outputs.go      # Response structures and converters
├── models/             # Data models for import tracking
│   └── models.go       # Import status and progress structures
├── routes/             # HTTP route handlers
│   └── routes.go       # Huma v2 route registrations
├── services/           # Business logic
│   └── service.go      # Import orchestration and Redis operations
├── module.go           # Module initialization and integration
└── CLAUDE.md           # This documentation
```

## SDE Data Import System

### Supported Data Types

The module can import all SDE data types supported by the `pkg/sde` service:

- **agents**: Mission agents with location and corporation info
- **categories**: Item categories with internationalized names
- **blueprints**: Manufacturing blueprints with material requirements
- **marketGroups**: Market categorization and hierarchy
- **metaGroups**: Item meta group classifications
- **npcCorporations**: NPC corporation data with faction info
- **typeIDs**: Basic type information (lightweight)
- **types**: Complete item type database with attributes
- **typeMaterials**: Manufacturing material requirements per type

### Redis Storage Structure

SDE data is stored in Redis using the following key patterns:

```
sde:agents:{agent_id}           # Agent data as JSON
sde:categories:{category_id}    # Category data as JSON
sde:blueprints:{blueprint_id}   # Blueprint data as JSON
sde:marketGroups:{group_id}     # Market group data as JSON
sde:metaGroups:{group_id}       # Meta group data as JSON
sde:npcCorporations:{corp_id}   # NPC corporation data as JSON
sde:typeIDs:{type_id}           # Basic type info as JSON
sde:types:{type_id}             # Full type data as JSON
sde:typeMaterials:{type_id}     # Type materials as JSON array

sde:metadata:last_import        # Timestamp of last successful import
```

### Import Process

1. **Initialization**: Create import status record with unique ID
2. **Data Loading**: Ensure SDE service is loaded from local files
3. **Batch Processing**: Import data in configurable batches (default: 1000 items)
4. **Progress Tracking**: Update real-time progress for each data type
5. **Error Handling**: Graceful error handling with detailed error messages
6. **Completion**: Mark import as completed with duration statistics

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
    "message": "SDE admin module is operational"
  }
}
```

### Administrative Endpoints

All administrative endpoints require **Super Administrator** permissions.

#### Start SDE Import
```
POST /sde_admin/import
```
**Authentication:** Super Admin Required
**Permission:** Super Administrator group membership

Start an import operation to load SDE data from files into Redis.

**Request Body:**
```json
{
  "data_types": ["agents", "types", "categories"],  // Optional: specific data types
  "force": false,                                   // Optional: overwrite existing data
  "batch_size": 1000                               // Optional: batch processing size
}
```

**Response:**
```json
{
  "body": {
    "import_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "pending",
    "message": "Import started for 3 data types",
    "start_time": "2024-01-15T10:30:00Z"
  }
}
```

**Parameters:**
- `data_types`: Array of data type names to import (empty = all types)
- `force`: Whether to overwrite existing Redis keys (default: false)
- `batch_size`: Number of items to process per batch (default: 1000, range: 100-10000)

#### Get Import Status
```
GET /sde_admin/import/{import_id}/status
```
**Authentication:** Super Admin Required

Get real-time status and progress of an import operation.

**Response:**
```json
{
  "body": {
    "import_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "running",
    "start_time": "2024-01-15T10:30:00Z",
    "duration": null,
    "progress": {
      "total_steps": 9,
      "completed_steps": 3,
      "current_step": "Processing types (4/9)",
      "percent_complete": 33.3,
      "data_types": {
        "agents": {
          "name": "agents",
          "status": "completed",
          "count": 15420,
          "processed": 15420,
          "percent_complete": 100.0
        },
        "types": {
          "name": "types",
          "status": "processing", 
          "count": 87451,
          "processed": 34500,
          "percent_complete": 39.4
        }
      }
    },
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:32:45Z"
  }
}
```

**Status Values:**
- `pending`: Import queued but not started
- `running`: Import in progress
- `completed`: Import finished successfully
- `failed`: Import failed with error

#### Get SDE Statistics
```
GET /sde_admin/stats
```
**Authentication:** Super Admin Required

Get statistics about SDE data currently stored in Redis.

**Response:**
```json
{
  "body": {
    "total_keys": 125843,
    "data_types": {
      "agents": {
        "count": 15420,
        "key_pattern": "sde:agents:*"
      },
      "types": {
        "count": 87451,
        "key_pattern": "sde:types:*"
      }
    },
    "last_import": "2024-01-15T10:35:22Z",
    "redis_memory_used": "245.7M"
  }
}
```

#### Clear SDE Data
```
DELETE /sde_admin/clear
```
**Authentication:** Super Admin Required

Remove all SDE data from Redis. **Use with caution - this cannot be undone.**

**Response:**
```json
{
  "body": {
    "success": true,
    "message": "Successfully deleted 125843 SDE keys",
    "keys_deleted": 125843
  }
}
```

## Progress Tracking System

### Import Status Model

Import operations are tracked with comprehensive status information:

```go
type ImportStatus struct {
    ID          string                // Unique import identifier
    Status      string               // pending, running, completed, failed
    StartTime   *time.Time           // When import started
    EndTime     *time.Time           // When import finished
    Progress    ImportProgress       // Detailed progress information
    Error       string               // Error message if failed
    CreatedAt   time.Time            // When import was created
    UpdatedAt   time.Time            // Last status update
}

type ImportProgress struct {
    TotalSteps     int                          // Total number of data types
    CompletedSteps int                          // Completed data types
    CurrentStep    string                       // Current operation description
    DataTypes      map[string]DataTypeStatus    // Per-type progress
}

type DataTypeStatus struct {
    Name      string    // Data type name
    Status    string    // pending, processing, completed, failed
    Count     int       // Total items to process
    Processed int       // Items processed so far
    Error     string    // Error message if failed
}
```

### Progress Storage

- **Active Imports**: Stored in memory (`sync.Map`) for fast access during import
- **Persistence**: All status updates saved to MongoDB `sde_import_status` collection
- **Cleanup**: Active imports removed from memory when completed/failed

### Real-time Updates

Progress is updated in real-time during import:

1. **Batch Updates**: Progress updated every batch (typically every 1000 items)
2. **Step Transitions**: Status updated when moving between data types
3. **Error Capture**: Detailed error information captured and stored
4. **Performance Metrics**: Duration and throughput statistics

## Error Handling

### Import Error Management

- **Data Type Errors**: Individual data type failures don't stop entire import
- **Batch Retry**: Failed batches are retried with exponential backoff
- **Graceful Degradation**: System continues operating if Redis is temporarily unavailable
- **Error Logging**: Comprehensive error logging with context information

### HTTP Error Responses

- **400 Bad Request**: Invalid request parameters (e.g., invalid data type names)
- **401 Unauthorized**: Missing or invalid authentication token
- **403 Forbidden**: User does not have Super Administrator permissions
- **404 Not Found**: Import ID not found
- **500 Internal Server Error**: System errors (Redis connection, file access, etc.)

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

### Batch Processing

- **Configurable Batch Size**: Default 1000 items, adjustable from 100-10000
- **Memory Efficiency**: Processes data in chunks to avoid memory exhaustion
- **Redis Pipeline**: Uses Redis pipelines for efficient bulk operations
- **Progress Updates**: Minimal database writes for progress tracking

### Import Performance

- **Expected Throughput**: 5000-10000 items per second (depending on data complexity)
- **Memory Usage**: ~50-200MB during import (depending on batch size)
- **Redis Storage**: ~100-500MB for complete SDE dataset
- **Duration**: Complete import typically takes 2-5 minutes

### Monitoring

- **Active Import Tracking**: Real-time monitoring of running imports
- **Memory Usage**: Redis memory usage reporting
- **Key Count Statistics**: Per-data-type key counts and patterns
- **Last Import Tracking**: Timestamp of most recent successful import

## Security Features

### Super Administrator Only

All administrative operations require Super Administrator group membership:
- Import operations cannot be initiated by regular users
- Progress monitoring restricted to administrators
- Data clearing requires highest privilege level
- Statistical data access limited to administrators

### Audit Trail

- **Operation Logging**: All administrative actions logged with operator information
- **Import History**: Complete history of import operations maintained
- **Error Logging**: Failed operations logged with detailed error information
- **Performance Metrics**: Import duration and throughput logged for analysis

## Integration Points

### SDE Service Integration

- **Data Source**: Loads data from `pkg/sde` service (file-based)
- **Service Validation**: Ensures SDE service is loaded before import
- **Type Compatibility**: Supports all SDE data types from the service
- **Error Propagation**: SDE service errors handled gracefully

### Redis Integration  

- **Key Management**: Systematic key naming for easy identification
- **Bulk Operations**: Efficient Redis pipeline operations
- **Memory Monitoring**: Redis memory usage tracking
- **Data Persistence**: No expiration times - data persists until cleared

### MongoDB Integration

- **Status Persistence**: Import status records stored for audit and recovery
- **Query Optimization**: Indexed collections for efficient status retrieval
- **Data Consistency**: Atomic updates for status information
- **Historical Data**: Maintains complete import history

## Future Enhancements

### Planned Features

- **Incremental Updates**: Update only changed SDE data
- **Scheduled Imports**: Automatic imports on SDE updates
- **Import Validation**: Verify data integrity after import
- **Export Functionality**: Export SDE data from Redis to files

### Advanced Features

- **Multi-Redis Support**: Import to multiple Redis instances
- **Data Compression**: Compress JSON data in Redis
- **Import Templates**: Pre-configured import profiles
- **Notification System**: Alert administrators on import completion/failure

## Dependencies

### Internal Dependencies

- `go-falcon/pkg/sde` (SDE service interface)
- `go-falcon/pkg/database` (MongoDB and Redis clients)
- `go-falcon/pkg/middleware` (Authentication and permissions)
- `go-falcon/pkg/handlers` (Standard response utilities)

### External Dependencies

- `github.com/danielgtaylor/huma/v2` (API framework)
- `github.com/google/uuid` (Unique ID generation)
- `go.mongodb.org/mongo-driver` (MongoDB operations)
- `github.com/go-redis/redis/v8` (Redis operations)

This SDE Admin module provides a comprehensive solution for managing EVE Online static data in Redis with full administrative control, real-time progress tracking, and robust error handling suitable for production environments.