# Groups and Site Settings Migration

## Overview

This migration implements the new **auto-join group system** based on enabled corporations and alliances. The system automatically assigns characters to groups when they authenticate, based on their corporation or alliance membership and the entities enabled in site settings.

## What This Migration Does

### 1. **Clean Slate Group System**
- **DROPS** all existing `groups` and `group_memberships` collections
- Creates new collections with updated schema supporting EVE entity integration
- ⚠️ **ALL EXISTING GROUP DATA WILL BE LOST**

### 2. **New Group Structure**
The new group model supports:
- **EVE Entity Groups**: Groups tied to specific corporations or alliances
- **Ticker-based Naming**: Groups named like `corp_TICKER` and `alliance_TICKER`
- **Auto-Creation**: Groups created automatically when entities are enabled
- **Auto-Assignment**: Characters automatically join relevant groups on login

### 3. **Site Settings Integration**
- Creates `managed_corporations` setting for corporation management
- Creates `managed_alliances` setting for alliance management  
- Each entity includes `ticker` field for group naming
- Enable/disable functionality for controlling group creation

### 4. **System Groups**
Creates three essential system groups:
- **Super Administrator** (`super_admin`): Full system access
- **Authenticated Users** (`authenticated`): Basic authenticated access  
- **Guest Users** (`guest`): Unauthenticated access

## New Group Naming Convention

| Entity Type | Naming Format | Example |
|-------------|---------------|---------|
| Corporation | `corp_TICKER` | `corp_BRAVE` |
| Alliance | `alliance_TICKER` | `alliance_BRAVE` |
| System | `system_name` | `super_admin` |
| Custom | User defined | `fleet_commanders` |

## How Auto-Join Works

1. **Entity Configuration**: Admins add corporations/alliances via Site Settings API
2. **Group Creation**: Enabling an entity automatically creates its group
3. **Character Login**: When a character authenticates:
   - System checks their corporation and alliance
   - If corp/alliance is enabled, character joins the corresponding group
   - Previous group memberships from disabled entities are removed

## Database Schema Changes

### Groups Collection (`groups`)
```javascript
{
  "_id": ObjectId,
  "name": "corp_BRAVE",                    // Group display name
  "description": "Corporation group for Brave Newbies Inc.",
  "type": "corporation",                   // system, corporation, alliance, custom
  "system_name": "super_admin",           // For system groups only
  "eve_entity_id": 99005338,              // Corporation or Alliance ID
  "eve_entity_ticker": "BRAVE",           // Entity ticker for naming
  "eve_entity_name": "Brave Newbies Inc.", // Full entity name
  "is_active": true,
  "created_at": ISODate,
  "updated_at": ISODate
}
```

### Site Settings Collections (`site_settings`)
```javascript
// Managed Corporations Setting
{
  "key": "managed_corporations",
  "value": {
    "corporations": [
      {
        "corporation_id": 99005338,
        "name": "Brave Newbies Inc.",
        "ticker": "BRAVE",                 // NEW: Used for group naming
        "enabled": true,
        "position": 1,
        "added_at": ISODate,
        "added_by": 12345,
        "updated_at": ISODate,
        "updated_by": 12345
      }
    ]
  },
  "type": "object",
  "category": "eve"
}

// Managed Alliances Setting  
{
  "key": "managed_alliances",
  "value": {
    "alliances": [
      {
        "alliance_id": 99005338,
        "name": "Brave Collective",  
        "ticker": "BRAVE",                 // NEW: Used for group naming
        "enabled": true,
        "position": 1,
        "added_at": ISODate,
        "added_by": 12345,
        "updated_at": ISODate,
        "updated_by": 12345
      }
    ]
  },
  "type": "object", 
  "category": "eve"
}
```

## Running the Migration

### Prerequisites
1. **Database Access**: Ensure MongoDB is running and accessible
2. **Configuration**: Valid `.env` file with `MONGO_URI` and `MONGO_DATABASE`
3. **Backup**: Take a database backup before running (recommended)

### Step-by-Step Process

1. **Navigate to project root**:
   ```bash
   cd /path/to/go-falcon
   ```

2. **Run the migration**:
   ```bash
   ./scripts/run_migration.sh
   ```

3. **Follow prompts**: The script will ask for confirmation before proceeding

4. **Verify completion**: Check the console output for success messages

### Manual Alternative
If the shell script doesn't work, you can run the migration manually:

```bash
# Build the migration binary
go build -o ./tmp/migrate_groups_and_site_settings ./scripts/migrate_groups_and_site_settings.go

# Run the migration
./tmp/migrate_groups_and_site_settings

# Clean up
rm ./tmp/migrate_groups_and_site_settings
```

## Post-Migration Setup

### 1. **Add Managed Entities**
Use the Site Settings API to add corporations and alliances:

```bash
# Add a corporation
curl -X POST http://localhost:3000/site-settings/corporations \
  -H "Authorization: Bearer <super_admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "corporation_id": 99005338,
    "name": "Brave Newbies Inc.",
    "ticker": "BRAVE",
    "enabled": true
  }'

# Add an alliance  
curl -X POST http://localhost:3000/site-settings/alliances \
  -H "Authorization: Bearer <super_admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "alliance_id": 99005338,
    "name": "Brave Collective",
    "ticker": "BRAVE", 
    "enabled": true
  }'
```

### 2. **Verify Group Creation**
Check that groups were created automatically:

```bash
# List all groups
curl -X GET http://localhost:3000/groups \
  -H "Authorization: Bearer <token>"
```

### 3. **Test Character Login**
Have a character from an enabled corporation/alliance log in and verify they're automatically added to the correct group.

## API Changes

### New Site Settings Endpoints
- `POST /site-settings/corporations` - Add managed corporation
- `PUT /site-settings/corporations/{id}/status` - Enable/disable corporation
- `GET /site-settings/corporations` - List managed corporations
- `POST /site-settings/alliances` - Add managed alliance
- `PUT /site-settings/alliances/{id}/status` - Enable/disable alliance  
- `GET /site-settings/alliances` - List managed alliances

### Enhanced Group Endpoints
- All group endpoints now support the new schema with EVE entity fields
- Groups are automatically created when entities are enabled
- Group memberships are automatically managed during authentication

## Troubleshooting

### Migration Fails to Start
- **Check database connection**: Verify `MONGO_URI` and `MONGO_DATABASE` in `.env`
- **Check permissions**: Ensure the user has read/write access to the database
- **Check Go version**: Ensure Go 1.19+ is installed

### Migration Partially Completes
- **Database locks**: Ensure no other processes are accessing the database
- **Network issues**: Check database connectivity during migration
- **Re-run safe**: The migration uses upsert operations and is safe to re-run

### Post-Migration Issues
- **Groups not created**: Verify entities are marked as `enabled: true`
- **Auto-join not working**: Check auth service integration and character context middleware
- **Missing permissions**: Ensure the first user is assigned to `super_admin` group

## Rollback Strategy

⚠️ **This migration cannot be automatically rolled back** because it drops existing collections.

### Manual Rollback Options:
1. **Database Restore**: Restore from a pre-migration backup
2. **Selective Restore**: Recreate only the collections that were dropped:
   - Restore `groups` collection from backup
   - Restore `group_memberships` collection from backup
   - Remove the new site settings entries

## Security Considerations

### Super Admin Assignment
- **First User Auto-Assignment**: The first user to authenticate after migration is automatically assigned to the `Super Administrator` group
- **Manual Assignment**: Additional super admins must be added via the Groups API
- **Group-Based**: Super admin status is now determined by group membership, not profile flags

### Data Protection
- **Clean Slate**: All existing group assignments are lost
- **Audit Trail**: New group memberships include creation timestamps and user attribution
- **Permission Validation**: All group operations require proper authentication

## Integration Points

### Auth Service
- Enhanced character context resolution with corporation/alliance data
- Auto-join logic triggered during authentication and profile refresh
- Backward compatible authentication flow

### Groups Service  
- New interface for site settings integration
- Auto-join methods for character group assignment
- Clean slate group membership management

### Site Settings Service
- New enabled entity management methods
- Ticker field support for group naming
- Enhanced corporation and alliance management

## Benefits of This Migration

1. **Automated Management**: Groups automatically reflect organizational structure
2. **Real-time Updates**: Group memberships update when characters change corps/alliances
3. **Simplified Administration**: Enable/disable entities instead of managing individual memberships
4. **Consistent Naming**: Standardized group names based on entity tickers
5. **Scalable Architecture**: Clean separation between entity management and group system

## Support

If you encounter issues with this migration:

1. **Check logs**: Review the migration output for error messages
2. **Verify configuration**: Ensure environment variables are correctly set
3. **Database state**: Check that collections were created properly
4. **API testing**: Test the new endpoints to ensure they're working
5. **Character flow**: Test the complete authentication and auto-join flow

For additional support, refer to the main project documentation or submit an issue with:
- Migration output/error messages
- Database configuration (sanitized)
- Go version and environment details