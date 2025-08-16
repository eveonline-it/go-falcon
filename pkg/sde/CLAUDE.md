# SDE Package (pkg/sde)

## Overview
EVE Online Static Data Export (SDE) in-memory service providing ultra-fast access to game static data including agents, categories, blueprints, types, and market information. Single instance shared across all modules.

## Core Features
- **In-Memory Storage**: All data loaded at startup for nanosecond access
- **Type-Safe Access**: Structured Go types with proper JSON unmarshaling  
- **Lazy Loading**: Data loaded on first access to optimize startup
- **Thread-Safe**: Concurrent access via read-write mutexes
- **Extensible**: Easy to add new SDE data types

## Data Sources
- **Source Files**: `data/sde/*.json` converted from CCP's YAML format
- **Processing System**: Web-based SDE management via `internal/sde` module
- **Update Process**: Automated detection and web-based management of new static data

## Available Data Types
- **Agents**: Mission agents with location and corporation info
- **Categories**: Item categories with internationalized names
- **Blueprints**: Manufacturing blueprints with material requirements
- **Market Groups**: Market categorization and hierarchy
- **Meta Groups**: Item meta group classifications
- **NPC Corporations**: NPC corporation data with faction info
- **Types**: Complete item type database with attributes
- **Type Materials**: Manufacturing material requirements

## Performance Characteristics
- **Memory Usage**: ~50-500MB depending on data size
- **Access Speed**: O(1) map lookups, O(log n) sorted searches
- **Startup Impact**: 1-2 second initial load time
- **No Network Calls**: All data served from memory

## Usage Examples
```go
// Direct data access
agent, err := sdeService.GetAgent("3008416")
category, err := sdeService.GetCategory("1")
blueprint, err := sdeService.GetBlueprint("1000001")

// Query operations
agents := sdeService.GetAgentsByLocation(60000004)
publishedTypes := sdeService.GetPublishedTypes()
marketGroups := sdeService.GetAllMarketGroups()
```

## Service Interface
```go
type SDEService interface {
    IsLoaded() bool
    GetAgent(agentID string) (*Agent, error)
    GetCategory(categoryID string) (*Category, error)
    GetBlueprint(blueprintID string) (*Blueprint, error)
    // ... many more data access methods
}
```

## Integration Points
- **Module Access**: Available through base module interface
- **ESI Enrichment**: Combines with live ESI data
- **Development Testing**: Used by dev module for validation
- **Query Operations**: Supports complex data queries and filtering

## Thread Safety
- **Read-Write Mutexes**: Safe concurrent access
- **Immutable Data**: No modifications after loading
- **Lazy Initialization**: Thread-safe first-time loading
- **Shared Instance**: Single service across all modules