# SDE Module (internal/sde)

## Overview

The SDE (Static Data Export) module provides web-based management of EVE Online SDE updates with background processing, progress tracking, and notification system. It offers a fully integrated web-controlled system for managing EVE Online static data.

## Architecture

### Core Components

- **Module**: Main SDE module with HTTP API endpoints
- **Background Processing**: Automated download, conversion, and storage
- **Progress Tracking**: Real-time progress updates stored in Redis
- **Hash Verification**: MD5 hash checking for update detection
- **Redis JSON Storage**: SDE data stored as individual Redis JSON keys for granular access
- **Scheduler Integration**: Background update checking via system tasks

### Files Structure

```
internal/sde/
├── sde.go          # Main module with API handlers and update logic
├── types.go        # Type definitions and data structures
├── utils.go        # Utility functions for file processing
└── CLAUDE.md       # This documentation file
```

## Features

### SDE Management
- **Update Detection**: Automatic checking for new SDE versions via hash comparison
- **Web Interface**: RESTful API for manual update initiation and status monitoring
- **Background Processing**: Non-blocking SDE downloads and conversions
- **Progress Tracking**: Real-time progress reporting with detailed status messages
- **Force Updates**: Manual override for re-processing current SDE version

### Data Processing
- **Download**: Automated download of SDE zip files from CCP servers
- **Extraction**: Zip file extraction with progress tracking
- **YAML to JSON**: Conversion of YAML files to JSON format
- **Redis JSON Storage**: Individual Redis JSON keys for each SDE entity with granular access
- **File Management**: Temporary file handling with cleanup

### Integration
- **Scheduler**: Background checking every 6 hours via system task
- **Notification**: Alerts when new versions are available
- **Status Persistence**: Redis-based status storage for distributed access
- **Module System**: Full integration with the gateway's module architecture

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/sde/status` | GET | Get current SDE status and version information |
| `/sde/check` | POST | Check for new SDE versions manually |
| `/sde/update` | POST | Start SDE update process |
| `/sde/progress` | GET | Get real-time update progress |
| `/sde/health` | GET | Module health check |
| `/sde/entity/{type}/{id}` | GET | Get individual SDE entity by type and ID |
| `/sde/entities/{type}` | GET | Get all entities of a specific type |
| `/sde/test/store-sample` | POST | Store sample test data for development |
| `/sde/test/verify` | GET | Verify individual key storage functionality |

### Status Response Format
```json
{
  "current_hash": "abc123...",
  "latest_hash": "def456...",
  "is_up_to_date": false,
  "is_processing": false,
  "progress": 0.0,
  "last_error": "",
  "last_check": "2024-01-01T12:00:00Z",
  "last_update": "2024-01-01T10:00:00Z"
}
```

### Update Request Format
```json
{
  "force_update": false
}
```

### Progress Response Format
```json
{
  "is_processing": true,
  "progress": 0.75,
  "message": "Converting YAML files...",
  "error": ""
}
```

### Individual Entity Access

**Get Single Entity:**
```bash
curl http://localhost:8080/sde/entity/agents/3008416
```

**Response:**
```json
{
  "agentTypeID": 2,
  "corporationID": 1000002,
  "divisionID": 1,
  "isLocator": false,
  "level": 1,
  "locationID": 60000004,
  "quality": 0
}
```

**Get All Entities by Type:**
```bash
curl http://localhost:8080/sde/entities/categories
```

**Response:**
```json
{
  "1": {
    "name": {"en": "System"},
    "published": true
  },
  "2": {
    "name": {"en": "Celestial"},
    "published": true
  }
}
```

## Background Processing

### Update Workflow
1. **Hash Check**: Compare current hash with latest from CCP
2. **Download**: Download SDE zip file with progress tracking
3. **Extract**: Extract zip contents to temporary directory
4. **Convert**: Convert YAML files to JSON format
5. **Cleanup**: Remove old SDE data from Redis
6. **Store**: Save processed data as individual Redis JSON keys
7. **Finalize**: Update status and cleanup temporary files

### Processed SDE Files
- `agents.yaml` → `agents.json`
- `blueprints.yaml` → `blueprints.json`
- `categories.yaml` → `categories.json`
- `marketGroups.yaml` → `marketGroups.json`
- `metaGroups.yaml` → `metaGroups.json`
- `npcCorporations.yaml` → `npcCorporations.json`
- `types.yaml` → `types.json`
- `typeDogma.yaml` → `typeDogma.json`
- `typeMaterials.yaml` → `typeMaterials.json`

### Progress Stages
- **0.1**: Download started
- **0.3**: Download completed, extraction started
- **0.5**: Extraction completed, conversion started
- **0.7**: Conversion completed, cleanup and Redis storage started
- **0.9**: Individual key storage completed, finalizing
- **1.0**: Update completed successfully

## Scheduler Integration

### System Task: `system-sde-check`
- **Schedule**: Every 6 hours (`0 0 */6 * * *`)
- **Priority**: Normal
- **Function**: Checks for new SDE versions
- **Auto-Update**: Disabled by default (check only)
- **Notifications**: Enabled when updates are available

### Task Configuration
```json
{
  "task_name": "sde_check",
  "parameters": {
    "auto_update": false,
    "notify": true
  }
}
```

### Integration Method
The scheduler calls `CheckSDEUpdate(ctx)` method which:
1. Fetches latest hash from CCP
2. Compares with current stored hash
3. Sends notifications if update available
4. Optionally triggers automatic update

## Redis JSON Storage

### Individual Key Structure
Instead of storing entire datasets, each SDE entity is stored as an individual Redis JSON key:

**Pattern:** `sde:{type}:{id}`

**Examples:**
- `sde:agents:3008416` → Individual agent object
- `sde:agents:3008417` → Another agent object  
- `sde:categories:1` → Category object for ID 1
- `sde:blueprints:1000001` → Blueprint object for ID 1000001
- `sde:types:587` → Type object for ID 587

### Metadata Keys
- `sde:current_hash`: Current SDE version hash
- `sde:status`: JSON-encoded status information
- `sde:progress`: Real-time progress data

### Storage Benefits
- **Granular Access**: Retrieve individual entities without loading entire datasets
- **Memory Efficiency**: Load only needed data
- **Better Caching**: Individual TTLs and cache strategies per entity
- **Parallel Processing**: Concurrent access to different entities
- **Redis JSON Features**: Leverage JSON path queries and partial updates

### Data Access Methods
SDE data can be accessed through:
- **Individual Entity API**: `GET /sde/entity/{type}/{id}`
- **Bulk Type API**: `GET /sde/entities/{type}`
- **Direct Redis Access**: Using Redis JSON commands
- **pkg/sde Service**: Loads from `data/sde/*.json` files (legacy)

## Notification System

### Update Available Notification
```go
func (m *Module) sendUpdateNotification(status *SDEStatus) {
    slog.Info("New SDE version available", 
        "current", status.CurrentHash,
        "latest", status.LatestHash)
}
```

### Completion Notification
```go
func (m *Module) sendCompletionNotification(status *SDEStatus) {
    slog.Info("SDE update completed",
        "hash", status.CurrentHash,
        "time", status.LastUpdate)
}
```

### Future Enhancements
- WebSocket notifications for real-time UI updates
- Email/Discord notifications for administrators
- Integration with notification module for user alerts

## Error Handling

### Common Error Scenarios
- Network failures during download
- Invalid or corrupted SDE files
- Redis storage failures
- Disk space issues during processing
- Concurrent update attempts

### Error Recovery
- Automatic retry for transient failures
- Graceful degradation on storage issues
- Cleanup of temporary files on errors
- Status preservation across failures

## Configuration

### Environment Variables
- CCP SDE URLs are hardcoded constants
- Redis configuration inherited from base module
- Progress tracking settings configurable

### Constants
```go
const (
    sdeURL          = "https://eve-static-data-export.s3-eu-west-1.amazonaws.com/tranquility/sde.zip"
    sdeHashURL      = "https://eve-static-data-export.s3-eu-west-1.amazonaws.com/tranquility/checksum"
    redisHashKey    = "sde:current_hash"
    redisStatusKey  = "sde:status"
    redisProgressKey = "sde:progress"
)
```

## Performance Considerations

### Memory Usage
- Temporary files stored on disk during processing
- Redis storage for processed data
- Minimal memory footprint during operation
- Cleanup of temporary data after processing

### Network Usage
- SDE downloads ~100-500MB depending on content
- Hash checks are lightweight (32 bytes)
- Progress updates to Redis are minimal

### Storage Requirements
- **Temporary**: ~1GB during processing  
- **Redis**: ~50-500MB for individual JSON keys (same total size, different structure)
- **Persistent**: JSON files in `data/sde/` directory

## Development Notes

### Modern SDE Management
The SDE module provides comprehensive web-based management:
- **Web Interface**: REST API for all SDE operations
- **Automated Processing**: Background scheduling and execution
- **Benefits**: Real-time status, progress tracking, full integration

### Core Functions
Key functions for SDE management:
- `unzipFile()`: Archive extraction with progress tracking
- `convertYAMLToJSON()`: YAML to JSON format conversion
- `downloadFileWithProgress()`: HTTP downloads with progress reporting
- `storeSDEFileAsIndividualKeys()`: Store SDE data as individual Redis JSON keys
- `GetSDEEntityFromRedis()`: Retrieve single entity by type and ID
- `GetSDEEntitiesByType()`: Retrieve all entities of a specific type
- `CleanupOldSDEData()`: Remove old SDE keys before updates

### Thread Safety
- Mutex protection for concurrent operations
- Redis for distributed state management
- Safe handling of background processing

## Testing

### Manual Testing
```bash
# Check SDE status
curl http://localhost:8080/sde/status

# Check for updates
curl -X POST http://localhost:8080/sde/check

# Start update
curl -X POST http://localhost:8080/sde/update \
  -H "Content-Type: application/json" \
  -d '{"force_update": false}'

# Monitor progress
curl http://localhost:8080/sde/progress

# Test individual key storage
curl -X POST http://localhost:8080/sde/test/store-sample
curl http://localhost:8080/sde/test/verify

# Access individual entities
curl http://localhost:8080/sde/entity/agents/3008416
curl http://localhost:8080/sde/entities/categories
```

### Integration Testing
- Scheduler task execution
- Redis data verification
- Progress tracking accuracy
- Error handling scenarios

## Future Enhancements

### Planned Features
- **Differential Updates**: Only update changed files
- **Rollback Support**: Ability to revert to previous version
- **Validation**: Verify data integrity after processing
- **Compression**: Compress Redis data for efficiency
- **Metrics**: Detailed update statistics and timing

### API Enhancements
- WebSocket endpoint for real-time progress
- Batch operations for multiple SDE versions
- Export functionality for processed data
- Administrative endpoints for maintenance

### Integration Improvements
- **Notification Module**: Rich notification support
- **Metrics**: Integration with monitoring systems
- **Audit Log**: Track all SDE operations
- **Auto-Update**: Configurable automatic updates