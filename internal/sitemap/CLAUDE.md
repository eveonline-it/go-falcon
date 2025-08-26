# Sitemap Module (internal/sitemap)

## Overview

The sitemap module provides **backend-managed dynamic routing and navigation** for the go-falcon EVE Online API gateway. This system gives the backend complete control over which routes users can access in the frontend, based on their permissions and group memberships. The module integrates seamlessly with the existing groups and permissions systems to provide secure, role-based access to frontend routes.

**Key Architecture**:
- **Flat Routes Array**: A simple list of all available routes for React Router configuration (no nesting/children)
- **Hierarchical Navigation**: A nested tree structure with folders for rendering the vertical navigation menu
- **Separation of Concerns**: Routes define what's available, Navigation defines how it's organized visually

**Frontend Location**: The React 19 frontend is located at `~/react-falcon` (`/home/tore/react-falcon`)

**Status**: Production Ready - Complete Backend Implementation
**Integration**: Full integration with groups, permissions, and auth systems
**Architecture**: Backend-controlled dynamic routing with React consumer APIs

## Architecture

### Core Concept

The sitemap module implements a **backend-first routing system** where:
- **Backend defines routes**: All frontend routes are stored and managed in MongoDB
- **Permission-based access**: Routes are filtered by user permissions and groups
- **Dynamic navigation**: Navigation menus are generated based on accessible routes
- **React consumption**: Frontend consumes route configurations via REST APIs
- **Real-time updates**: Route access can change without frontend deployments

### Response Structure Design

The module provides two distinct data structures in its response:
1. **Routes Array**: A flat list of all available routes (no nesting) that React Router uses to configure routing
2. **Navigation Tree**: A hierarchical structure with folders that the UI uses to render the vertical menu

This separation allows the frontend to:
- Configure routing simply with the flat routes array
- Build complex nested navigation menus using the hierarchical navigation tree
- Keep routing logic separate from menu presentation

### Files Structure

```
internal/sitemap/
├── dto/
│   ├── inputs.go          # Request input DTOs with Huma v2 validation
│   └── outputs.go         # Response output DTOs for all endpoints
├── models/
│   └── models.go         # MongoDB schemas and response structures
├── routes/
│   └── routes.go         # Huma v2 route definitions and handlers
├── services/
│   ├── service.go        # Business logic for route filtering and management
│   └── repository.go     # Database operations and MongoDB queries
├── module.go             # Module initialization and route seeding
└── CLAUDE.md             # This documentation

**Note**: Authentication and permission middleware now centralized in `pkg/middleware/` with `SitemapAdapter` for module-specific methods.
```

## Data Models

### Route Schema (MongoDB Collection: `routes`)

```go
type Route struct {
    ID                  primitive.ObjectID    `bson:"_id,omitempty"`
    RouteID            string                `bson:"route_id"`         // Frontend identifier
    Path               string                `bson:"path"`             // React Router path
    Component          string                `bson:"component"`        // React component name
    Name               string                `bson:"name"`             // Display name
    Icon               *string               `bson:"icon,omitempty"`   // FontAwesome icon
    Type               RouteType             `bson:"type"`             // public|auth|protected|admin
    ParentID           *string               `bson:"parent_id"`        // For nested routes
    
    // Navigation Configuration
    NavPosition        NavigationPosition    `bson:"nav_position"`     // main|user|admin|footer|hidden
    NavOrder           int                   `bson:"nav_order"`        // Sort order
    ShowInNav          bool                  `bson:"show_in_nav"`      // Visibility in nav
    
    // Access Control
    RequiredPermissions []string             `bson:"required_permissions"` // AND logic
    RequiredGroups     []string              `bson:"required_groups"`      // OR logic
    
    // Metadata
    Title              string                `bson:"title"`            // Page title
    Description        *string               `bson:"description"`      // SEO description
    Keywords           []string              `bson:"keywords"`         // Search keywords
    Group              *string               `bson:"group"`            // UI grouping
    
    // React Integration
    Props              map[string]interface{} `bson:"props"`           // Component props
    LazyLoad           bool                  `bson:"lazy_load"`        // Code splitting
    Exact              bool                  `bson:"exact"`            // Exact path matching
    NewTab             bool                  `bson:"newtab"`           // Open in new tab
    
    // Feature Flags
    FeatureFlags       []string              `bson:"feature_flags"`    // Required features
    IsEnabled          bool                  `bson:"is_enabled"`       // Route enabled
    
    // Audit
    CreatedAt          time.Time             `bson:"created_at"`
    UpdatedAt          time.Time             `bson:"updated_at"`
}
```

### Route Types

- **`public`**: No authentication required (landing page, login)
- **`auth`**: Authentication required only (profile, basic features)
- **`protected`**: Specific permissions required (admin panels, sensitive data)
- **`admin`**: Super administrator access only (system management)

### Navigation Positions

- **`main`**: Primary navigation sidebar
- **`user`**: User dropdown menu
- **`admin`**: Admin panel navigation
- **`footer`**: Footer links
- **`hidden`**: Accessible but not in navigation menus

## API Endpoints

### User Endpoints (Authentication Required)

#### Get User Sitemap
```http
GET /sitemap
Authorization: Bearer <token>
```

**Description**: Returns personalized routes and navigation for the authenticated user

**Query Parameters**:
- `include_disabled` (boolean): Include disabled routes (default: false)
- `include_hidden` (boolean): Include hidden navigation items (default: false)

**Response**:
```json
{
  "body": {
    "routes": [
      {
        "id": "dashboard-analytics",
        "path": "/dashboard/analytics", 
        "component": "AnalyticsDashboard",
        "name": "Analytics",
        "title": "Analytics Dashboard",
        "permissions": ["analytics.view"],
        "lazyLoad": true,
        "accessible": true,
        "meta": {
          "title": "Analytics Dashboard",
          "icon": "chart-pie",
          "group": "dashboard"
        }
      }
    ],
    "navigation": [
      {
        "label": "main",
        "labelDisable": true,
        "children": [
          {
            "routeId": "folder-dashboard",
            "name": "Dashboard",
            "icon": "folder",
            "isFolder": true,
            "hasChildren": true,
            "children": [
              {
                "routeId": "dashboard-analytics",
                "name": "Analytics",
                "to": "/dashboard/analytics",
                "icon": "chart-pie",
                "active": true
              }
            ]
          }
        ]
      }
    ],
    "userPermissions": ["analytics.view", "dashboard.access"],
    "userGroups": ["Authenticated Users"],
    "features": {
      "darkMode": true,
      "advancedAnalytics": true
    }
  }
}
```

**Key Response Structure**:
- **`routes`**: Flat array of available routes (no children/nesting) for React Router configuration
- **`navigation`**: Hierarchical tree structure with folders for rendering the vertical navigation menu

#### Check Route Access
```http
GET /sitemap/access/{route_id}
Authorization: Bearer <token>
```

**Description**: Check if current user can access a specific route

**Response**:
```json
{
  "body": {
    "route_id": "dashboard-analytics",
    "path": "/dashboard/analytics",
    "accessible": true,
    "reason": "",
    "missing": []
  }
}
```

### Public Endpoints (No Authentication)

#### Get Public Sitemap
```http
GET /sitemap/public
```

**Description**: Returns only public routes for unauthenticated users and SEO

**Response**:
```json
{
  "body": {
    "routes": [...], 
    "navigation": [...]
  }
}
```

#### Module Status
```http
GET /sitemap/status
```

**Description**: Health check for the sitemap module

### Admin Endpoints (Super Admin Required)

#### List All Routes
```http
GET /admin/sitemap
Authorization: Bearer <token>
```

**Query Parameters**:
- `type`: Filter by route type (public|auth|protected|admin)
- `group`: Filter by route group
- `is_enabled`: Filter by enabled status
- `show_in_nav`: Filter by navigation visibility
- `nav_position`: Filter by navigation position
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20, max: 100)

#### Create Route
```http
POST /admin/sitemap
Authorization: Bearer <token>
Content-Type: application/json

{
  "route_id": "dashboard-analytics",
  "path": "/dashboard/analytics",
  "component": "AnalyticsDashboard", 
  "name": "Analytics",
  "type": "protected",
  "required_permissions": ["analytics.view", "dashboard.access"],
  "nav_position": "main",
  "nav_order": 10,
  "show_in_nav": true,
  "title": "Analytics Dashboard",
  "group": "dashboard",
  "lazy_load": true,
  "is_enabled": true,
  "icon": "chart-pie"
}
```

#### Update Route
```http
PUT /admin/sitemap/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Updated Analytics",
  "is_enabled": false
}
```

#### Delete Route
```http
DELETE /admin/sitemap/{id}
Authorization: Bearer <token>
```

**Note**: Deletes the route and all its children recursively

#### Bulk Reorder Navigation
```http
POST /admin/sitemap/reorder
Authorization: Bearer <token>
Content-Type: application/json

{
  "updates": [
    {"route_id": "dashboard-analytics", "nav_order": 5},
    {"route_id": "dashboard-crm", "nav_order": 10}
  ]
}
```

#### Get Route Statistics
```http
GET /admin/sitemap/stats
Authorization: Bearer <token>
```

**Response**:
```json
{
  "body": {
    "total_routes": 45,
    "enabled_routes": 42,
    "disabled_routes": 3,
    "public_routes": 5,
    "protected_routes": 25,
    "routes_by_type": {
      "public": 5,
      "auth": 15,
      "protected": 20,
      "admin": 5
    },
    "routes_by_group": {
      "dashboard": 8,
      "app": 12,
      "admin": 10
    }
  }
}
```

## Integration with Groups and Permissions

### Permission-Based Filtering

Routes are filtered using a **hybrid approach**:

1. **Permission Requirements (AND logic)**:
   ```json
   {
     "required_permissions": ["analytics.view", "dashboard.access"]
   }
   ```
   User must have ALL listed permissions

2. **Group Requirements (OR logic)**:
   ```json
   {
     "required_groups": ["Super Administrator", "Analytics Team"]
   }
   ```
   User must belong to ANY listed group

3. **Combined Logic**:
   Access granted if user has (ALL required permissions) OR (ANY required group)

### Super Admin Bypass

Users in the "Super Administrator" group automatically get access to all routes, regardless of specific permission requirements.

### Multi-Character Support

The system supports multi-character users by:
- Aggregating permissions across all user characters
- Granting access based on the union of all character permissions
- Maintaining user context through the existing auth system

## Frontend Integration (~/react-falcon)

### Understanding the Response Structure

The sitemap API returns two separate data structures:

1. **`routes`** - Flat array for React Router configuration
   - No nested children
   - Each route is independent
   - Used to configure React Router paths

2. **`navigation`** - Hierarchical tree for menu rendering
   - Contains folders and nested items
   - Used to build the vertical navigation menu
   - Supports multiple levels of nesting

### React Router Integration

The frontend at `~/react-falcon` consumes the sitemap API to:

1. **Generate Routes Dynamically** (using flat routes array):
   ```typescript
   // hooks/useSitemap.ts
   export function useSitemap() {
     return useQuery({
       queryKey: ['sitemap'],
       queryFn: () => api.get<SitemapResponse>('/sitemap'),
       staleTime: 5 * 60 * 1000,
     });
   }
   
   // App.tsx - Use flat routes array for React Router
   const { data: sitemap } = useSitemap();
   const routes = sitemap?.routes || []; // Flat array, no children
   ```

2. **Build Navigation Structure** (using hierarchical navigation):
   ```typescript
   // components/Navigation.tsx
   const { data: sitemap } = useSitemap();
   
   return (
     <nav>
       {sitemap?.navigation.map(group => (
         <NavGroup key={group.label} group={group} />
       ))}
     </nav>
   );
   ```

3. **Protect Routes**:
   ```typescript
   // App.tsx - Dynamic route generation from flat array
   const router = createDynamicRouter(sitemap.routes); // No nested children
   return <RouterProvider router={router} />;
   ```

### Component Registry

The frontend maintains a component registry matching backend route definitions:

```typescript
// components/routing/RouteRegistry.js
const routeComponents = {
  'AnalyticsDashboard': lazy(() => import('demos/dashboards/AnalyticsDashboard')),
  'CrmDashboard': lazy(() => import('demos/dashboards/CrmDashboard')),
  'Chat': lazy(() => import('features/chat/Chat')),
  'UserProfile': lazy(() => import('pages/user/profile/Profile')),
  // ... more components
};
```

### Frontend Implementation Files

Based on the existing structure at `~/react-falcon`, these files support dynamic routing:

- **`FRONTEND_DYNAMIC_ROUTING.md`**: Implementation plan (already exists)
- **`BACKEND_DYNAMIC_ROUTING.md`**: Backend specification (already exists)
- **`src/hooks/useAuthRoutes.js`**: Hook for fetching user routes
- **`src/components/routing/`**: Dynamic routing components
- **`src/routes/siteMaps.ts`**: Static routes being migrated to dynamic

## Database Indexes

The repository automatically creates optimized indexes:

```javascript
// Compound indexes for optimal query performance
{
  "route_id": 1,                                    // Unique identifier
  "type": 1,                                        // Route type filtering
  "is_enabled": 1,                                  // Enabled filtering
  "nav_position": 1, "nav_order": 1,              // Navigation queries
  "parent_id": 1,                                   // Hierarchical queries
  "group": 1,                                       // Grouping queries
  "is_enabled": 1, "type": 1, "nav_position": 1    // Combined filtering
}
```

## Permission Registration

The module registers its own permissions during initialization:

```go
sitemapPermissions := []permissions.Permission{
    {
        ID:          "sitemap:routes:view",
        Service:     "sitemap", 
        Resource:    "routes",
        Action:      "view",
        Name:        "View Routes",
        Description: "View route configurations",
        Category:    "Sitemap Management",
    },
    {
        ID:          "sitemap:routes:manage",
        Service:     "sitemap",
        Resource:    "routes", 
        Action:      "manage",
        Name:        "Manage Routes",
        Description: "Create, update, and delete route configurations",
        Category:    "Sitemap Management",
    },
    // ... more permissions
}
```

## Default Route Seeding

### Automated Seeding

The module includes `SeedDefaultRoutes()` method that populates the database with routes matching the existing React frontend structure at `~/react-falcon`:

```go
// Seed routes matching src/routes/siteMaps.ts structure
func (m *Module) SeedDefaultRoutes(ctx context.Context) error {
    defaultRoutes := m.getDefaultRoutes() // Based on ~/react-falcon structure
    
    for _, route := range defaultRoutes {
        if !routeExists {
            m.service.CreateRoute(ctx, &route)
        }
    }
}
```

### Pre-configured Routes

Default routes include:

**Dashboard Routes**:
- `/` → DefaultDashboard (auth required)
- `/dashboard/analytics` → AnalyticsDashboard (protected: analytics.view)
- `/dashboard/crm` → CrmDashboard (protected: crm.view)
- `/dashboard/saas` → SaasDashboard (protected: saas.view)

**Application Routes**:
- `/app/calendar` → Calendar (auth required)
- `/app/chat` → Chat (auth required)
- `/app/kanban` → Kanban (protected: kanban.view)
- `/app/email/inbox` → EmailInbox (protected: email.view)

**User Routes**:
- `/user/profile` → UserProfile (auth required)
- `/user/characters` → Characters (auth required)

**Admin Routes** (admin type):
- `/admin/users` → UsersAdmin
- `/admin/groups` → GroupsAdmin
- `/admin/permissions` → PermissionsAdmin
- `/admin/scheduler` → SchedulerAdmin

**Public Routes**:
- `/landing` → Landing (public)

## Error Handling

### HTTP Status Codes
- **200 OK**: Successful operation
- **401 Unauthorized**: Authentication required
- **403 Forbidden**: Insufficient permissions  
- **404 Not Found**: Route not found
- **400 Bad Request**: Invalid input or constraint violation
- **500 Internal Server Error**: Database or system error

### Error Response Format
```json
{
  "error": "error_code",
  "message": "Human-readable error message", 
  "details": "Additional context (optional)"
}
```

### Graceful Degradation

- **Database Unavailable**: Frontend falls back to cached routes
- **Permission Service Down**: Routes fall back to group-based checking
- **Invalid Component**: Frontend shows 404 instead of crashing
- **Network Issues**: Frontend retries with exponential backoff

## Security Considerations

### Access Control
- **Route-level Security**: Each route protected by permissions/groups
- **Admin Endpoint Protection**: All admin operations require super admin access
- **Input Validation**: Comprehensive validation on all input DTOs
- **SQL Injection Prevention**: MongoDB parameterized queries throughout

### Audit Trail
- **Route Changes**: All create/update/delete operations logged
- **Access Attempts**: Failed route access attempts logged
- **Permission Changes**: Route permission modifications tracked
- **User Actions**: All admin operations include user context

## Performance Optimizations

### Database Performance
- **Optimized Indexes**: Compound indexes for common query patterns
- **Aggregation Pipelines**: Efficient permission checking queries
- **Connection Pooling**: MongoDB connection reuse
- **Query Limits**: Pagination prevents large result sets

### Frontend Performance  
- **Route Caching**: 5-minute client-side route cache
- **Lazy Loading**: Components loaded on-demand
- **Navigation Pruning**: Only accessible routes sent to frontend
- **Minimized Payload**: Route filtering reduces JSON size

### Caching Strategy
```go
// Future: Redis caching for frequently accessed routes
type CacheStrategy struct {
    UserRoutesTTL    time.Duration // 5 minutes
    PublicRoutesTTL  time.Duration // 1 hour
    PermissionsTTL   time.Duration // 10 minutes
}
```

## Monitoring and Observability

### Health Checks
- **Module Status**: `/sitemap/status` endpoint
- **Database Connectivity**: MongoDB connection testing
- **Route Statistics**: Usage and performance metrics
- **Permission Integration**: Groups/permissions service health

### Metrics
- Route access frequency
- Permission check performance
- Navigation generation time
- Frontend route cache hit rates

### Logging
```go
// Structured logging throughout
slog.Info("[Sitemap] Route created", 
    "route_id", routeID,
    "user_id", userID, 
    "permissions", permissions)
```

## Development Workflow

### Adding New Routes

1. **Backend Route Creation**:
   ```bash
   POST /admin/sitemap
   {
     "route_id": "new-feature",
     "path": "/features/new",
     "component": "NewFeature",
     "type": "protected",
     "required_permissions": ["feature.view"]
   }
   ```

2. **Frontend Component Registration**:
   ```typescript
   // Add to RouteRegistry.js
   'NewFeature': lazy(() => import('features/NewFeature'))
   ```

3. **Permission Setup**:
   ```go
   // Register permissions in module
   "feature:view:access": "View New Feature"
   ```

### Testing Routes

```bash
# Test user route access
curl -H "Authorization: Bearer $TOKEN" \
     "localhost:3000/sitemap/access/new-feature"

# Test admin route management  
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
     "localhost:3000/admin/sitemap"
```

## Migration from Static Routes

### Phase 1: Parallel Operation
1. Keep existing static routes in `~/react-falcon/src/routes/`
2. Add dynamic route fetching alongside static routes
3. Test with limited user group

### Phase 2: Gradual Migration
1. Move high-security routes to backend control first
2. Migrate admin routes and protected features
3. Update navigation components

### Phase 3: Complete Migration  
1. Remove static route definitions
2. Full dynamic route consumption
3. Cleanup unused frontend routing code

### Migration Helper
```bash
# Seed existing routes from frontend
POST /admin/sitemap/seed-from-frontend
# Analyzes ~/react-falcon/src/routes/siteMaps.ts and creates routes
```

## Troubleshooting

### Common Issues

**Routes Not Appearing**:
- Check user permissions with `/sitemap/access/{route_id}`
- Verify route is enabled: `is_enabled: true`
- Ensure user is authenticated

**Permission Errors**:
- Verify permissions exist in system
- Check group membership for user
- Confirm route permission requirements

**Frontend Integration Issues**:
- Check component registry has matching component name
- Verify API_PREFIX environment variable alignment
- Ensure route paths match React Router expectations

### Debug Commands
```bash
# Check route statistics
GET /admin/sitemap/stats

# Verify user permissions
GET /sitemap (with user token)

# Test specific route access
GET /sitemap/access/{route_id}
```

## Future Enhancements

### Planned Features
- **Route Analytics**: Track which routes are most accessed
- **A/B Testing**: Conditional route exposure for testing
- **Route Scheduling**: Time-based route activation
- **Localization**: Multi-language route names and descriptions
- **Route Templates**: Pre-configured route sets for common roles

### Advanced Features
- **Conditional Routes**: Context-based route availability
- **Route Versioning**: Multiple versions of routes for gradual rollouts
- **Real-time Updates**: WebSocket-based route updates
- **Route Dependencies**: Routes that require other routes to be accessible

## Dependencies

### Internal Dependencies
- `go-falcon/internal/auth` (authentication and user models)
- `go-falcon/internal/groups` (group membership resolution)  
- `go-falcon/pkg/permissions` (permission checking)
- `go-falcon/pkg/handlers` (HTTP response utilities)
- `go-falcon/pkg/module` (module interface)

### External Dependencies
- `github.com/danielgtaylor/huma/v2` (API framework)
- `go.mongodb.org/mongo-driver` (MongoDB driver)
- `go.mongodb.org/mongo-driver/bson` (BSON encoding)

### Frontend Dependencies (~/react-falcon)
- `@tanstack/react-query` (API state management)
- `react-router-dom` (routing)
- `zustand` (state management)

## Contributing

### Code Standards
1. Follow existing Go Falcon module patterns
2. Use Huma v2 for all new endpoints
3. Implement comprehensive input validation
4. Add proper error handling and logging
5. Update documentation for any changes
6. Include tests for new functionality

### Testing Requirements
- Unit tests for all service methods
- Integration tests for route filtering logic
- API endpoint tests with authentication
- Frontend integration tests with mock backend
- Performance tests for large route sets

## Support

### Documentation
- **Backend**: This CLAUDE.md file
- **Frontend**: `~/react-falcon/FRONTEND_DYNAMIC_ROUTING.md`
- **Integration**: `~/react-falcon/BACKEND_DYNAMIC_ROUTING.md`

### Common Questions
1. **Q: How do I add a new protected route?**
   A: Use POST /admin/sitemap with required_permissions array

2. **Q: Why isn't my route showing in navigation?**
   A: Check show_in_nav: true and nav_position is not "hidden"

3. **Q: How do I update navigation order?**
   A: Use POST /admin/sitemap/reorder with nav_order updates

4. **Q: Can I have nested routes?**  
   A: Yes, use parent_id field to create hierarchical routes

This sitemap module provides complete backend control over frontend routing, enabling secure, permission-based access to application features while maintaining excellent performance and user experience. The integration with `~/react-falcon` creates a seamless dynamic routing system that scales with your application's security requirements.

## Important Implementation Notes

- **Routes are flat**: The `routes` array in the response contains no nested children - it's a simple flat list
- **Navigation is hierarchical**: The `navigation` array contains folders and nested items for menu rendering  
- **Folders are navigation-only**: Folder entries exist only in navigation, not in the routes array
- **Separation of concerns**: Routes define what's accessible, navigation defines how it's organized visually