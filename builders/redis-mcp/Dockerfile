FROM python:3.12-alpine

# Install git and uv for Python package management
RUN apk add --no-cache git && pip install uv

# Install Redis MCP server from GitHub
RUN uvx --from git+https://github.com/redis/mcp-redis.git redis-mcp-server --help || true

# Set working directory
WORKDIR /app

# Default command with environment variable support
ENTRYPOINT ["uvx", "--from", "git+https://github.com/redis/mcp-redis.git", "redis-mcp-server"]