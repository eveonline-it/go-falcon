# MCP Servers Configuration for Claude Code CLI

## Overview
MCP (Model Context Protocol) servers enable direct database access from Claude Code CLI for querying collections, inspecting data, and real-time monitoring.

### Available MCP Servers
- **MongoDB MCP** - Access to CASBIN collections, user profiles, and application data
- **Redis MCP** - Access to cache data, sessions, and key-value storage
- **Rewatch MCP** - Background process management for development servers
- **CORE Memory MCP** - Persistent cross-platform AI memory and context management

## Docker Setup (✅ Running)

### MongoDB MCP Server
- Container: `go-falcon-mongodb-mcp`
- Connection: `mongodb://admin:password123@mongodb:27017/falcon?authSource=admin`
- Status: ✅ Working
- Network: `go-falcon_falcon-network`

### Redis MCP Server
- Container: `go-falcon-redis-mcp` 
- Connection: `redis://redis:6379/0`
- Status: ✅ Working (Docker implementation)
- Network: `go-falcon_falcon-network`

### Rewatch MCP Server
- Type: Node.js process manager
- Config: `rewatch.config.json`
- Status: ✅ Working (manages backend/infra processes)
- Purpose: Background development server management

### CORE Memory MCP Server
- Type: AI memory and context management system
- URL: https://github.com/RedPlanetHQ/core
- Purpose: Cross-platform persistent memory for AI conversations
- Features: Knowledge graph, temporal context, multi-tool sync

## Claude Code CLI Configuration (✅ Completed)

### Add MCP Servers via Command Line

For Claude Code CLI, use the `claude mcp add` command instead of manual config files:

#### MongoDB MCP Server (✅ Working)
```bash
claude mcp add mongodb-falcon -- docker exec -i go-falcon-mongodb-mcp mongodb-mcp-server
```

#### Redis MCP Server (✅ Working)
```bash
# Docker approach (working!)
claude mcp add redis-falcon -- docker exec -i go-falcon-redis-mcp uvx --from git+https://github.com/redis/mcp-redis.git redis-mcp-server --url redis://redis:6379/0

# Alternative: Local NPX approach (if preferred)
# claude mcp add redis-local --env REDIS_URI=redis://localhost:6379/0 -- npx -y @redis/mcp-redis
```

#### Rewatch MCP Server (✅ Working)
```bash
# Add Rewatch MCP server for process management
claude mcp add rewatch -- npx -y mcp-rewatch
```

#### CORE Memory MCP Server
```bash
# Add CORE Memory MCP server for AI context management
# First sign up at: https://core.heysol.ai
# Then get your API key and add the server
claude mcp add core-memory -- npx -y @redplanethq/core-memory-server
```

**Setup Requirements:**
1. Create account at https://core.heysol.ai
2. Generate API key from dashboard
3. Configure environment variables:
   ```bash
   export CORE_API_KEY="your-api-key"
   export CORE_USER_ID="your-user-id"
   ```

### Verify Configuration

Check that MCP servers are connected:

```bash
claude mcp list
```

Expected output:
```
mongodb-falcon: docker exec -i go-falcon-mongodb-mcp mongodb-mcp-server - ✓ Connected
redis-falcon: docker exec -i go-falcon-redis-mcp uvx --from git+https://github.com/redis/mcp-redis.git redis-mcp-server --url redis://redis:6379/0 - ✓ Connected
rewatch: npx -y mcp-rewatch - ✓ Connected
core-memory: npx -y @redplanethq/core-memory-server - ✓ Connected
```

### Configuration Files

Claude Code CLI stores MCP configurations in:
- **Project level**: `/home/tore/.claude.json` (current configuration)
- **User level**: `~/.config/claude-code/mcp_servers.json`
- **Local level**: `./.mcp.json` (project-specific, can be committed to git)

### Rewatch Configuration

The Rewatch MCP server uses `rewatch.config.json` in the project root:

```json
{
  "processes": {
    "backend": {
      "command": "make",
      "args": ["dev"],
      "cwd": "./"
    },
    "infra": {
      "command": "docker",
      "args": [
        "compose",
        "-f",
        "docker-compose.infra.yml",
        "up"
      ],
      "cwd": "./"
    }
  }
}
```

#### Available Rewatch Tools
- **`list_processes`** - Show status of all configured processes
- **`restart_process`** - Restart a specific process by name
- **`get_process_logs`** - Retrieve recent logs from a process
- **`stop_all`** - Stop all running processes gracefully

#### Usage Examples
```bash
# Via Claude Code CLI (MCP tools automatically available)
# List all processes
mcp__rewatch__list_processes

# Start infrastructure
mcp__rewatch__restart_process --name infra

# Start backend server  
mcp__rewatch__restart_process --name backend

# Get recent logs
mcp__rewatch__get_process_logs --name backend --lines 50
```

#### CORE Memory Tools (Available with Setup)
- **`search`** - Search through persistent memory and context
- **`ingest`** - Store new information in the knowledge graph
- **`recall`** - Retrieve specific context or memories
- **`contextualize`** - Add context to current conversation

#### Usage Examples
```bash
# Via Claude Code CLI (MCP tools automatically available)
# Search for project context
mcp__core-memory__search --query "go-falcon authentication system"

# Store important information
mcp__core-memory__ingest --message "Updated super admin system to use database instead of env vars"

# Ask about previous work
mcp__core-memory__search --query "previous discussions about EVE Online integration"
```

### Rewatch Process Management (✅ Available)
With the Rewatch MCP server connected, process control capabilities:

#### Development Server Management
- **Process Control** - Start/stop/restart development servers
- **Log Monitoring** - Retrieve process output and error logs
- **Status Tracking** - Monitor running process states
- **Background Execution** - Keep processes running without blocking CLI

#### Go-Falcon Processes (from rewatch.config.json)
- **backend** - Main Go API gateway server (`make dev`)
- **infra** - Docker infrastructure services (`docker compose -f docker-compose.infra.yml up`)

### CORE Memory Management (Available with Setup)
With the CORE Memory MCP server connected, AI context capabilities:

#### Cross-Platform Memory
- **Knowledge Graph** - Persistent memory across AI tools (Claude, Cursor, ChatGPT, etc.)
- **Conversation History** - Synchronized context between different AI platforms
- **Project Context** - Shared understanding of codebase and development state
- **Temporal Tracking** - Time-based context with provenance information

#### Integration Features
- **Browser Extension** - Save conversations and context automatically
- **Multi-Tool Sync** - Context follows you between Cursor, VSCode, Claude Code
- **API Integration** - Connect with Linear, Slack, Notion, GitHub for context
- **Query Interface** - Ask "What do you know about X?" across platforms

## Troubleshooting

### Check MCP Connection Status
```bash
claude mcp list
```

### Check MCP Server Logs
```bash
docker logs go-falcon-mongodb-mcp
```

### Test Direct MongoDB Connection
```bash
docker exec go-falcon-mongodb mongosh -u admin -p password123 --authenticationDatabase admin --eval "db.adminCommand('listDatabases')"
```

### Verify Containers are Running
```bash
docker ps | grep falcon
```

### Common Issues

1. **MCP Server Not Connected**: Restart Docker containers or re-add MCP server
2. **Connection Refused**: Check if MongoDB container is running and accessible
3. **Permission Denied**: Verify MongoDB authentication credentials
4. **Container Not Found**: Make sure `go-falcon-mongodb-mcp` container exists

### Remove and Re-add MCP Server
```bash
# Remove existing server
claude mcp remove mongodb-falcon

# Re-add with correct configuration
claude mcp add mongodb-falcon -- docker exec -i go-falcon-mongodb-mcp mongodb-mcp-server
```

### Restart Infrastructure
```bash
docker compose -f docker-compose.infra.yml down
docker compose -f docker-compose.infra.yml up -d
```