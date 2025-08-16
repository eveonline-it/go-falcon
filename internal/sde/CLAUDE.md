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
- **Comprehensive YAML Discovery**: Recursive scanning of `bsd`, `fsd`, and `universe` directories for all YAML files
- **Universe Directory Filtering**: Selective processing of universe subdirectories excluding landmarks
- **YAML to JSON**: Conversion of all discovered YAML files to JSON format
- **Redis JSON Storage**: Individual Redis JSON keys for each SDE entity with granular access
- **File Management**: Temporary file handling with cleanup
- **Dynamic Processing**: Automatically adapts to new SDE files without code changes

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
| `/sde/search/solarsystem` | GET | Search for solar systems by name (query param: name) |
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

**Search Solar Systems by Name:**
```bash
curl "http://localhost:8080/sde/search/solarsystem?name=Jita"
```

**Response:**
```json
{
  "query": "Jita",
  "count": 1,
  "results": [
    {
      "systemName": "Jita",
      "region": "TheForge",
      "constellation": "Kimotoro",
      "universeType": "eve",
      "redisKey": "sde:universe:eve:TheForge:Kimotoro:Jita",
      "solarSystemID": 30000142,
      "security": 0.945913116665,
      "solarSystemNameID": 269099
    }
  ]
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
The SDE module now processes **ALL** YAML files found in both the `bsd` and `fsd` directories recursively, including:

**Core FSD Files:**
- `agents.yaml` → `agents.json`
- `blueprints.yaml` → `blueprints.json`
- `categories.yaml` → `categories.json`
- `marketGroups.yaml` → `marketGroups.json`
- `metaGroups.yaml` → `metaGroups.json`
- `npcCorporations.yaml` → `npcCorporations.json`
- `types.yaml` → `types.json`
- `typeDogma.yaml` → `typeDogma.json`
- `typeMaterials.yaml` → `typeMaterials.json`

**Additional Files:**
- All other YAML files in `fsd/` subdirectories
- All YAML files in `bsd/` directory and subdirectories
- Universe directory files from: `abyssal`, `eve`, `hidden`, `void`, and `wormhole` subdirectories
- Dynamic processing ensures all EVE Online SDE data is captured

**Universe Directory Structure:**
The universe directory follows a strict hierarchical structure:
```
universe/{type}/{region}/{constellation}/{system}/
```

**Processed Universe Types:**
- `universe/abyssal/{region}/{constellation}/{system}/` - Abyssal space data
- `universe/eve/{region}/{constellation}/{system}/` - New Eden universe data
- `universe/hidden/{region}/{constellation}/{system}/` - Hidden regions data
- `universe/void/{region}/{constellation}/{system}/` - Void space data  
- `universe/wormhole/{region}/{constellation}/{system}/` - Wormhole space data
- **Excluded**: `universe/landmarks/` (not processed per requirements)

**YAML File Locations:**
- **Region YAML**: Located in region directories (e.g., `region.yaml`)
- **Constellation YAML**: Located in constellation directories (e.g., `constellation.yaml`)
- **System YAML**: Located in system directories (e.g., `solarsystem.yaml`, other objects)

**Processing Features:**
- **Recursive Discovery**: Automatically finds all `.yaml` and `.yml` files
- **Comprehensive Directory Scanning**: Processes `bsd`, `fsd`, and `universe` directories
- **Universe Filtering**: Selectively processes only `abyssal`, `eve`, `hidden`, `void`, and `wormhole` subdirectories
- **Hierarchical Structure Support**: Handles region/constellation universe data structure
- **Dynamic Naming**: JSON output files maintain original YAML base names
- **Multi-Format Support**: Handles both map-based and array-based YAML structures
- **Smart ID Extraction**: Automatically identifies suitable ID fields including universe-specific IDs
- **Error Tolerance**: Failed file conversions don't stop the entire process
- **Progress Tracking**: Real-time progress updates show file conversion status

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

## Data Format Handling

### Supported YAML Structures

The SDE module automatically handles different YAML data structures found in EVE Online SDE files:

**Map-Based Format** (Key-Value pairs):
```yaml
1000001:
  activities:
    1:
      materials:
        - materialTypeID: 587
          quantity: 1
  blueprintTypeID: 1000001
  maxProductionLimit: 1
```

**Array-Based Format** (List of objects):
```yaml
- flagID: 0
  flagName: None
  flagText: None
  orderID: 0
- flagID: 1
  flagName: Wallet
  flagText: Wallet
  orderID: 10
```

### ID Extraction Strategy

For array-based data, the system automatically extracts suitable IDs using:

1. **Primary ID Fields**: `flagID`, `typeID`, `itemID`, `groupID`, etc.
2. **Secondary ID Fields**: `corporationID`, `systemID`, `blueprintID`, etc.
3. **Fallback Strategy**: Uses first available numeric/string field
4. **Index Fallback**: Uses array index if no suitable ID found

**Generated Redis Keys Examples:**
- Map data: `sde:blueprints:1000001`
- Array data with flagID: `sde:flags:0`, `sde:flags:1`
- Universe region file: `sde:universe:eve:Derelik`
- Universe constellation file: `sde:universe:eve:Derelik:Kador`
- Universe system file: `sde:universe:eve:Derelik:Kador:Amarr`
- Array data without ID: `sde:unknowntype:index_0`

## Universe Data Processing

### Hierarchical Structure Handling

The SDE module processes universe data respecting the EVE Online universe hierarchy while storing complete files:

**File Path Processing:**
1. **Discovery**: Walk through `universe/{type}/{region}/{constellation}/{system}/` structure
2. **Path Preservation**: Maintain hierarchical context during conversion
3. **Filename Generation**: Create descriptive filenames like `universe_eve_10000001_20000001_30000001_solarsystem.json`
4. **Complete File Storage**: Store entire YAML file content as single Redis JSON entries

**Data Level Classification:**
- **Region Level**: Files directly in region directories (e.g., `region.yaml`)
- **Constellation Level**: Files in constellation directories (e.g., `constellation.yaml`)  
- **System Level**: Files in system directories (e.g., `solarsystem.yaml`, objects)

**Redis Key Structure (Complete Files):**
```
sde:universe:{type}:{region}                               # Region data file
sde:universe:{type}:{region}:{constellation}               # Constellation data file  
sde:universe:{type}:{region}:{constellation}:{system}      # System data file
```

**Benefits:**
- **Complete Data Access**: Retrieve entire file content in single operation
- **Hierarchical Organization**: Spatial hierarchy preserved in Redis keys
- **Simplified Storage**: No data splitting, maintains original file structure
- **Easy Querying**: Direct access to complete region/constellation/system data

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
- `sde:universe:eve:Derelik:Kador:Amarr` → Complete Amarr system file

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
- `collectYAMLFiles()`: Recursively discover all YAML files in directories
- `collectUniverseYAMLFiles()`: Hierarchical collection from universe subdirectories with type filtering
- `collectUniverseTypeFiles()`: Walk universe type directories respecting region/constellation/system structure
- `convertYAMLFiles()`: Process all discovered YAML files from bsd, fsd, and universe directories
- `convertYAMLToJSON()`: Individual YAML to JSON format conversion with path context preservation
- `generateJSONFileName()`: Generate context-aware filenames for universe data preserving hierarchy
- `downloadFileWithProgress()`: HTTP downloads with progress reporting
- `determineDataTypeAndContext()`: Extract universe hierarchical context from filenames
- `storeSDEFileAsIndividualKeysWithContext()`: Context-aware storage for universe data
- `generateUniverseRedisKey()`: Generate hierarchical Redis keys for universe data
- `storeSDEFileAsIndividualKeys()`: Store SDE data as individual Redis JSON keys (supports both map and array formats)
- `extractEntityID()`: Smart ID extraction including universe-specific IDs (systemID, constellationID, etc.)
- `GetSDEEntityFromRedis()`: Retrieve single entity by type and ID
- `GetSDEEntitiesByType()`: Retrieve all entities of a specific type
- `CleanupOldSDEData()`: Remove all old SDE keys before updates (dynamic cleanup)

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

# Search solar systems
curl "http://localhost:8080/sde/search/solarsystem?name=Jita"
curl "http://localhost:8080/sde/search/solarsystem?name=Ama"  # Partial match
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
- **Selective Processing**: Option to process only specific directories or file patterns
- **Schema Validation**: Validate YAML structure before processing

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