# MCP Servers Configuration for Claude Code CLI

## Overview

MCP (Model Context Protocol) servers enable direct database access from Claude Code CLI for querying collections, inspecting data, and real-time monitoring.

### Available MCP Servers

- **MongoDB MCP** - Access to user profiles and application data
- **Redis MCP** - Access to cache data, sessions, and key-value storage

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

### Verify Configuration

Check that MCP servers are connected:

```bash
claude mcp list
```

Expected output:

```
mongodb-falcon: docker exec -i go-falcon-mongodb-mcp mongodb-mcp-server - ✓ Connected
redis-falcon: docker exec -i go-falcon-redis-mcp uvx --from git+https://github.com/redis/mcp-redis.git redis-mcp-server --url redis://redis:6379/0 - ✓ Connected
```

### Configuration Files

Claude Code CLI stores MCP configurations in:

- **Project level**: `/home/tore/.claude.json` (current configuration)
- **User level**: `~/.config/claude-code/mcp_servers.json`
- **Local level**: `./.mcp.json` (project-specific, can be committed to git)

#### Go-Falcon Processes (from rewatch.config.json)

- **backend** - Main Go API gateway server (`make dev`)
- **infra** - Docker infrastructure services (`docker compose -f docker-compose.infra.yml up`)

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
