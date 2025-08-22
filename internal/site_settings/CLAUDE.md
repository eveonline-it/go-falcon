# Site Settings Module (internal/site_settings)

## Overview

The site settings module provides centralized configuration management for the go-falcon EVE Online API gateway. This module enables super administrators to manage application-wide settings that control various aspects of the system's behavior, UI appearance, and feature availability.

**Current Status**: Production Ready - Full Super Admin Access Control
**Authentication**: Super admin group membership required for all management operations

## Architecture

### Core Components

- **Settings Management**: CRUD operations for site configuration settings
- **Corporation Management**: Complete CRUD operations for managed corporations with enable/disable functionality
- **Type Safety**: Strongly typed setting values with validation
- **Public/Private Settings**: Fine-grained visibility control
- **Category Organization**: Logical grouping of related settings
- **Audit Trail**: Complete history of setting changes and corporation management
- **Default Settings**: Predefined essential configuration settings

### Files Structure

```
internal/site_settings/
├── dto/
│   ├── inputs.go         # Request input DTOs with Huma v2 validation
│   └── outputs.go        # Response output DTOs
├── middleware/
│   └── auth.go           # Super admin authentication middleware
├── routes/
│   └── routes.go         # Huma v2 route definitions
├── services/
│   ├── service.go        # Business logic for settings management
│   └── repository.go     # Database operations and queries
├── models/
│   └── models.go         # MongoDB schemas and default settings
├── module.go             # Module initialization and interface implementation
└── CLAUDE.md             # This documentation
```

## Data Model

### Site Setting Schema (MongoDB Collection: `site_settings`)

```go
type SiteSetting struct {
    ID          primitive.ObjectID `bson:"_id,omitempty"`
    Key         string             `bson:"key"`                    // Unique setting identifier
    Value       interface{}        `bson:"value"`                  // Setting value (any type)
    Type        SettingType        `bson:"type"`                   // Data type validation
    Category    string             `bson:"category,omitempty"`     // Organization category
    Description string             `bson:"description,omitempty"`  // Human-readable description
    IsPublic    bool               `bson:"is_public"`              // Public visibility
    IsActive    bool               `bson:"is_active"`              // Active status
    CreatedBy   *int64             `bson:"created_by,omitempty"`   // Character ID who created
    UpdatedBy   *int64             `bson:"updated_by,omitempty"`   // Character ID who updated
    CreatedAt   time.Time          `bson:"created_at"`
    UpdatedAt   time.Time          `bson:"updated_at"`
}
```

### Setting Types

- **`string`**: Text-based configuration values
- **`number`**: Numeric values (integers and floats)
- **`boolean`**: True/false toggle settings
- **`object`**: Complex structured data (JSON objects)

### Setting Categories

- **`general`**: Basic site information and branding
- **`system`**: Core system behavior and limits
- **`auth`**: Authentication and user management
- **`api`**: API behavior and rate limiting
- **`eve`**: EVE Online integration settings
- **`notifications`**: Notification system configuration
- **`security`**: Security policies and restrictions
- **`ui`**: User interface customization

### Managed Corporations Data Model

The module stores managed corporations in a special site setting with key `"managed_corporations"` using the object type with proper BSON models for type-safe database operations. The structure is:

```json
{
  "key": "managed_corporations",
  "value": {
    "corporations": [
      {
        "corporation_id": 98000001,
        "name": "Example Corporation",
        "enabled": true,
        "added_at": "2025-01-01T00:00:00Z",
        "added_by": 12345,
        "updated_at": "2025-01-01T00:00:00Z",
        "updated_by": 12345
      }
    ]
  },
  "type": "object",
  "category": "eve",
  "description": "Managed corporations with enable/disable status"
}
```

**Corporation Fields:**
- `corporation_id` (int64): EVE Online corporation ID
- `name` (string): Corporation name
- `enabled` (boolean): Whether the corporation is enabled
- `added_at` (timestamp): When the corporation was first added
- `added_by` (int64): Character ID who added the corporation
- `updated_at` (timestamp): Last modification timestamp
- `updated_by` (int64): Character ID who made the last update

**Database Models:**
- `models.ManagedCorporation`: BSON-tagged struct for database operations
- `models.ManagedCorporationsValue`: Container structure for the corporations array
- Proper BSON marshaling/unmarshaling ensures accurate date and field handling

## Default Settings

The module automatically creates essential default settings on initialization:

| Setting Key | Type | Category | Public | Description |
|-------------|------|----------|--------|-------------|
| `site_name` | string | general | ✓ | The name displayed in the UI |
| `maintenance_mode` | boolean | system | ✓ | System maintenance status |
| `max_users` | number | system | ✗ | Maximum registered users |
| `api_rate_limit` | number | api | ✗ | API requests per minute per user |
| `registration_enabled` | boolean | auth | ✓ | New user registration availability |
| `contact_info` | object | general | ✓ | Administrator contact information |

## API Endpoints

### Public Endpoints (No Authentication Required)

#### Get Public Settings
```
GET /site-settings/public?category=general&page=1&limit=20
```
**Query Parameters:**
- `category` (optional): Filter by setting category
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 20, max: 100)

**Response:**
```json
{
  "settings": [
    {
      "key": "site_name",
      "value": "Go Falcon API Gateway",
      "type": "string",
      "category": "general",
      "description": "The name of the site displayed in the UI",
      "is_public": true,
      "is_active": true,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  ],
  "total": 10,
  "page": 1,
  "limit": 20,
  "total_pages": 1
}
```

### Protected Endpoints (Super Admin Only)

#### Create Setting
```
POST /site-settings
Authorization: Bearer <token> | Cookie: falcon_auth_token
```
**Request Body:**
```json
{
  "key": "new_setting",
  "value": "example_value",
  "type": "string",
  "category": "general",
  "description": "Example setting description",
  "is_public": true
}
```

#### List All Settings
```
GET /site-settings?category=system&is_public=false&page=1&limit=20
Authorization: Bearer <token> | Cookie: falcon_auth_token
```
**Query Parameters:**
- `category` (optional): Filter by category
- `is_public` (optional): Filter by public/private status
- `is_active` (optional): Filter by active/inactive status
- `page` (optional): Page number
- `limit` (optional): Items per page

#### Get Single Setting
```
GET /site-settings/{key}
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

#### Update Setting
```
PUT /site-settings/{key}
Authorization: Bearer <token> | Cookie: falcon_auth_token
```
**Request Body:**
```json
{
  "value": "updated_value",
  "description": "Updated description",
  "is_public": false,
  "is_active": true
}
```

#### Delete Setting
```
DELETE /site-settings/{key}
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

### Corporation Management Endpoints (Super Admin Only)

#### Add Managed Corporation
```
POST /site-settings/corporations
Authorization: Bearer <token> | Cookie: falcon_auth_token
```
**Request Body:**
```json
{
  "corporation_id": 98000001,
  "name": "Example Corporation",
  "enabled": true
}
```

**Response:**
```json
{
  "corporation": {
    "corporation_id": 98000001,
    "name": "Example Corporation",
    "enabled": true,
    "added_at": "2025-01-01T00:00:00Z",
    "added_by": 12345,
    "updated_at": "2025-01-01T00:00:00Z",
    "updated_by": 12345
  },
  "message": "Corporation 'Example Corporation' (ID: 98000001) added successfully"
}
```

#### List Managed Corporations
```
GET /site-settings/corporations?enabled=true&page=1&limit=20
Authorization: Bearer <token> | Cookie: falcon_auth_token
```
**Query Parameters:**
- `enabled` (optional): Filter by enabled status ('true', 'false', or empty for all)
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 20, max: 100)

**Response:**
```json
{
  "corporations": [
    {
      "corporation_id": 98000001,
      "name": "Example Corporation",
      "enabled": true,
      "added_at": "2025-01-01T00:00:00Z",
      "added_by": 12345,
      "updated_at": "2025-01-01T00:00:00Z",
      "updated_by": 12345
    }
  ],
  "total": 15,
  "page": 1,
  "limit": 20,
  "total_pages": 1
}
```

#### Get Specific Corporation
```
GET /site-settings/corporations/{corp_id}
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

#### Update Corporation Status
```
PUT /site-settings/corporations/{corp_id}/status
Authorization: Bearer <token> | Cookie: falcon_auth_token
```
**Request Body:**
```json
{
  "enabled": false
}
```

**Response:**
```json
{
  "corporation": {
    "corporation_id": 98000001,
    "name": "Example Corporation",
    "enabled": false,
    "added_at": "2025-01-01T00:00:00Z",
    "added_by": 12345,
    "updated_at": "2025-01-01T12:00:00Z",
    "updated_by": 12345
  },
  "message": "Corporation 'Example Corporation' (ID: 98000001) disabled successfully"
}
```

#### Remove Managed Corporation
```
DELETE /site-settings/corporations/{corp_id}
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

#### Bulk Update Corporations
```
PUT /site-settings/corporations
Authorization: Bearer <token> | Cookie: falcon_auth_token
```
**Request Body:**
```json
{
  "corporations": [
    {
      "corporation_id": 98000001,
      "name": "Example Corporation",
      "enabled": true
    },
    {
      "corporation_id": 98000002,
      "name": "Another Corporation",
      "enabled": false
    }
  ]
}
```

**Response:**
```json
{
  "corporations": [
    {
      "corporation_id": 98000001,
      "name": "Example Corporation",
      "enabled": true,
      "added_at": "2025-01-01T00:00:00Z",
      "added_by": 12345,
      "updated_at": "2025-01-01T12:00:00Z",
      "updated_by": 12345
    }
  ],
  "updated": 1,
  "added": 1,
  "message": "Bulk update completed: 1 corporations updated, 1 corporations added"
}
```

### Health Check
```
GET /site-settings/health
```

## Authentication and Authorization

### Permission Requirements

- **Public Endpoints**: No authentication required
- **All Management Operations**: Requires super admin group membership
- **Setting Creation/Update/Delete**: Super admin only
- **Private Setting Access**: Super admin only

### Super Admin Verification

The module uses the Character Context Middleware to verify super admin status:

```go
// Requires membership in "Super Administrator" system group
user, err := middleware.RequireSuperAdmin(ctx, authHeader, cookieHeader)
```

## Type Validation

The module provides comprehensive type validation for setting values:

### String Type
- Must be a valid string value
- Supports any text content

### Number Type  
- Accepts integers and floating-point numbers
- Supports string-to-number conversion
- Validates numeric format

### Boolean Type
- Must be true or false
- Strict boolean validation

### Object Type
- Accepts maps, slices, and structured data
- Supports complex JSON objects
- Validates object structure

## Database Operations

### Indexes

The module creates optimized database indexes for efficient queries:

- `key` (unique): Fast setting lookup
- `category`: Category-based filtering
- `is_public`: Public/private filtering
- `is_active`: Active status filtering
- `category + is_public`: Combined filtering

### Operations

- **Create**: Insert new settings with validation
- **Read**: Efficient key-based and filtered retrieval
- **Update**: Atomic updates with audit trail
- **Delete**: Safe setting removal
- **List**: Paginated queries with multiple filters

## Business Logic

### Setting Creation Rules
- Setting keys must be unique across the system
- Values must match specified type validation
- Audit trail records creator information
- Default settings created on module initialization

### Update Rules
- Type validation enforced on value updates
- Audit trail tracks modification history
- Supports partial updates of setting properties
- Maintains data integrity across updates

### Visibility Rules
- Public settings accessible without authentication
- Private settings require super admin access
- Fine-grained control over setting exposure
- Category-based organization for management

## Error Handling

### HTTP Status Codes
- **200 OK**: Successful operations
- **201 Created**: Setting created successfully
- **400 Bad Request**: Invalid input or validation errors
- **401 Unauthorized**: Authentication required
- **403 Forbidden**: Insufficient permissions (not super admin)
- **404 Not Found**: Setting not found
- **409 Conflict**: Setting key already exists
- **500 Internal Server Error**: Database or server errors

### Error Response Format
```json
{
  "status": 500,
  "title": "Internal Server Error",
  "detail": "Failed to create setting: validation error details"
}
```

## Integration Points

### Auth Module Integration
- JWT token validation for protected endpoints
- Character ID extraction for audit trails
- Super admin group membership verification
- Dual authentication support (Bearer tokens and cookies)

### Groups Module Integration  
- Super admin group membership checking
- Character context resolution with group data
- Permission validation against group membership
- Integration with group-based access control

### Application Integration
- Centralized configuration management
- Feature toggle capabilities
- UI customization settings
- System behavior control

## Performance Considerations

- **Database Indexes**: Optimized for common query patterns
- **Pagination**: All list endpoints support efficient pagination
- **Type Validation**: Client-side and server-side validation
- **Caching**: Prepared for future Redis caching implementation

## Development Workflow

### Adding New Settings
1. Define setting in `models/models.go` defaults if system-critical
2. Test setting creation via API
3. Implement any special validation logic
4. Update documentation with new setting details

### Adding New Setting Types
1. Add type to `SettingType` enum
2. Implement validation in `validateValueType` method
3. Update DTO validation rules
4. Add comprehensive tests for new type

## Security Considerations

- **Access Control**: Strict super admin requirement for management
- **Input Validation**: Comprehensive type and format validation
- **Audit Trail**: Complete change tracking with user attribution
- **Data Integrity**: Database constraints prevent invalid states
- **Type Safety**: Runtime type validation prevents data corruption

## Testing

### Testing Requirements
- Integration tests with real super admin authentication
- Type validation test scenarios for all setting types
- CRUD operation testing with error conditions
- Permission verification testing
- Database constraint and index testing

## Migration Path

### Database Migrations
- Site settings collection created automatically
- Default settings initialized on first run
- Database indexes created during module initialization
- No manual migration required for new deployments

## Configuration

### Environment Variables
- Currently uses standard database configuration
- No module-specific environment variables required

### Module Configuration
```go
// Module initialization
siteSettingsModule, err := site_settings.NewModule(db, authService, groupsService)
if err != nil {
    log.Fatal(err)
}

// Initialize with default settings
if err := siteSettingsModule.Initialize(ctx); err != nil {
    log.Fatal(err)
}

// Register routes
siteSettingsModule.RegisterUnifiedRoutes(api)
```

## Future Enhancements

### Planned Features
- Redis caching for improved read performance
- Setting validation schemas for complex objects
- Setting import/export functionality  
- Setting change notifications
- Advanced audit logging with detailed change history
- Setting templates and presets
- Bulk setting operations

### Advanced Features
- Setting dependency management
- Environment-specific setting overrides
- Setting rollback functionality
- Real-time setting updates via WebSocket
- Setting change approval workflows

## Usage Examples

### Managing Site Branding
```bash
# Update site name
curl -X PUT https://localhost:3000/site-settings/site_name \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"value": "My EVE Corp API", "is_public": true}'

# Get public settings for frontend
curl https://localhost:3000/site-settings/public?category=general
```

### System Configuration
```bash
# Enable maintenance mode
curl -X PUT https://localhost:3000/site-settings/maintenance_mode \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"value": true}'

# Update API rate limits  
curl -X PUT https://localhost:3000/site-settings/api_rate_limit \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"value": 200}'
```

### Corporation Management
```bash
# Add a new managed corporation
curl -X POST https://localhost:3000/site-settings/corporations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "corporation_id": 98000001,
    "name": "Test Corporation",
    "enabled": true
  }'

# List all managed corporations
curl -X GET https://localhost:3000/site-settings/corporations \
  -H "Authorization: Bearer <token>"

# List only enabled corporations
curl -X GET "https://localhost:3000/site-settings/corporations?enabled=true" \
  -H "Authorization: Bearer <token>"

# Disable a corporation
curl -X PUT https://localhost:3000/site-settings/corporations/98000001/status \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'

# Remove a corporation completely
curl -X DELETE https://localhost:3000/site-settings/corporations/98000001 \
  -H "Authorization: Bearer <token>"

# Bulk update corporations
curl -X PUT https://localhost:3000/site-settings/corporations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "corporations": [
      {
        "corporation_id": 98000001,
        "name": "Updated Corp Name",
        "enabled": true
      },
      {
        "corporation_id": 98000002,
        "name": "New Corporation",
        "enabled": false
      }
    ]
  }'

# Check if corporation is enabled (programmatic usage)
curl -X GET https://localhost:3000/site-settings/corporations/98000001 \
  -H "Authorization: Bearer <token>"
```

### Complex Settings
```bash
# Update contact information
curl -X PUT https://localhost:3000/site-settings/contact_info \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "value": {
      "email": "admin@mycorp.com",
      "discord": "https://discord.gg/mycorp",
      "website": "https://mycorp.com"
    },
    "type": "object"
  }'
```

## Dependencies

### Internal Dependencies
- `go-falcon/internal/auth/services` (authentication service)
- `go-falcon/internal/groups/services` (groups service for super admin)
- `go-falcon/internal/groups/middleware` (character context middleware)
- `go-falcon/pkg/database` (MongoDB connection)

### External Dependencies
- `github.com/danielgtaylor/huma/v2` (API framework)
- `go.mongodb.org/mongo-driver` (MongoDB driver)
- Standard Go libraries for type validation and JSON handling

## Contributing

1. Follow the established module structure pattern
2. Use Huma v2 for all new endpoints with proper validation
3. Implement comprehensive type validation for new setting types
4. Add proper error handling with meaningful messages
5. Update documentation for any changes
6. Include tests for new functionality
7. Maintain super admin access control requirements

## Monitoring and Observability

- **Structured Logging**: All operations logged with context
- **Health Checks**: Module health endpoint available
- **Error Tracking**: Comprehensive error logging and context
- **Audit Trail**: All setting changes tracked with user information
- **Performance Metrics**: Database operation tracking capability

This documentation reflects the production-ready implementation with full super admin access control and comprehensive site settings management capabilities.