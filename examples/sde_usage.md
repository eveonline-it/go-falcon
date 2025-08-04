# SDE Service Usage Examples

The SDE (Static Data Export) service provides fast, in-memory access to EVE Online static data.

## Available Endpoints

All endpoints are available under the `/dev/sde/` route in the development module:

### SDE Service Status
```bash
curl http://localhost:8080/api/v1/dev/sde/status
```

Response:
```json
{
  "source": "SDE Service",
  "status": "success",
  "data": {
    "loaded": true,
    "agents_count": 1234,
    "categories_count": 56,
    "blueprints_count": 789
  },
  "module": "dev",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Get Agent Information
```bash
curl http://localhost:8080/api/v1/dev/sde/agent/3008416
```

Response:
```json
{
  "source": "SDE Service",
  "status": "success",
  "data": {
    "agentTypeID": 2,
    "corporationID": 1000002,
    "divisionID": 22,
    "isLocator": false,
    "level": 1,
    "locationID": 60000004
  },
  "module": "dev",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Get Category Information
```bash
curl http://localhost:8080/api/v1/dev/sde/category/1
```

Response:
```json
{
  "source": "SDE Service",
  "status": "success",
  "data": {
    "name": {
      "de": "Besitzer",
      "en": "Owner",
      "es": "Propietario",
      "fr": "Propriétaire",
      "ja": "所有者",
      "ko": "소유자",
      "ru": "Владелец",
      "zh": "拥有者"
    },
    "published": true
  },
  "module": "dev",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Get Blueprint Information
```bash
curl http://localhost:8080/api/v1/dev/sde/blueprint/1000001
```

### Get Agents by Location
```bash
curl http://localhost:8080/api/v1/dev/sde/agents/location/60000004
```

Response:
```json
{
  "source": "SDE Service",
  "status": "success",
  "data": [
    {
      "agentTypeID": 2,
      "corporationID": 1000002,
      "divisionID": 22,
      "isLocator": false,
      "level": 1,
      "locationID": 60000004
    }
  ],
  "module": "dev",
  "count": 1,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Using SDE in Your Modules

Access SDE data from any module through the base module:

```go
func (m *YourModule) someHandler(w http.ResponseWriter, r *http.Request) {
    // Get agent information
    agent, err := m.SDEService().GetAgent("3008416")
    if err != nil {
        // Handle error
        return
    }
    
    // Get category with internationalization
    category, err := m.SDEService().GetCategory("1")
    if err != nil {
        // Handle error
        return
    }
    
    // Access localized names
    englishName := category.Name["en"]  // "Owner"
    germanName := category.Name["de"]   // "Besitzer"
    
    // Get agents by location
    agents, err := m.SDEService().GetAgentsByLocation(60000004)
    if err != nil {
        // Handle error
        return
    }
    
    // Use the data...
}
```

## Performance Characteristics

- **Data Loading**: Lazy loaded on first access
- **Memory Usage**: ~50-500MB depending on data size
- **Access Speed**: Nanosecond lookups (in-memory maps)
- **Thread Safety**: Concurrent access supported
- **No External Dependencies**: Direct memory access, no Redis/database calls

## Data Updates

To update SDE data:

1. Run the SDE conversion tool: `./sde`
2. Restart the gateway to reload data
3. Check status via `/dev/sde/status` endpoint