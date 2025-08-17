# CLAUDE.md - OpenAPI Expert Integration

## Overview

This document describes how to use the OpenAPI expert agent to maintain synchronization between your Golang REST API implementation and the OpenAPI specification, which is then exported as `falcon-openapi.json` for frontend consumption.

## Agent Purpose

The OpenAPI expert agent specializes in:
- Analyzing your Golang API implementation in `internal/api`
- Maintaining an accurate `openapi.yml` specification
- Ensuring complete API documentation coverage
- Facilitating the export to `falcon-openapi.json`

## Integration Workflow

### 1. Prerequisites

- Ensure your Golang project structure follows the expected layout:
  ```
  internal/
  ├── api/
  │   ├── controllers/
  │   ├── routes/
  │   ├── dto/
  │   │   ├── request/
  │   │   └── response/
  │   └── middleware/
  ```
- Have an existing `openapi.yml` file (or create one)
- Your Golang OpenAPI command should be configured to read from `openapi.yml`

### 2. Using the Agent

When you need to update your OpenAPI specification:

```bash
# 1. First, ask the agent to analyze your API
"Analyze all endpoints in internal/api and compare with openapi.yml"

# 2. Request specific updates
"Update openapi.yml to include the new user authentication endpoints"

# 3. After agent updates openapi.yml, run your export command
go run cmd/openapi/main.go export falcon-openapi.json
```

### 3. Agent Capabilities

The agent will:

- **Scan Route Definitions**: Identify all Gin routes, HTTP methods, and path parameters
- **Analyze DTOs**: Extract request/response schemas from struct definitions
- **Document Validation**: Convert Go validation tags to OpenAPI constraints
- **Security Schemes**: Document authentication and authorization requirements
- **Error Responses**: Ensure all error scenarios are properly documented

### 4. Export Command Integration

Your Golang command should:

1. Read the updated `openapi.yml`
2. Validate the specification
3. Transform to JSON format
4. Export as `falcon-openapi.json`

Example command structure:
```go
// cmd/openapi/main.go
func main() {
    // Read openapi.yml
    spec := readOpenAPISpec("openapi.yml")
    
    // Validate against actual implementation
    validateSpec(spec, scanAPIEndpoints())
    
    // Export to JSON
    exportJSON(spec, "falcon-openapi.json")
}
```

### 5. Automated Workflow

Consider creating a script that combines both steps:

```bash
#!/bin/bash
# update-api-docs.sh

echo "Updating OpenAPI specification..."
# Use the agent to update openapi.yml
# (This would be done through your Claude interface)

echo "Exporting to falcon-openapi.json..."
go run cmd/openapi/main.go export falcon-openapi.json

echo "Validating exported specification..."
go run cmd/openapi/main.go validate falcon-openapi.json

echo "API documentation updated successfully!"
```

## Best Practices

### 1. Regular Synchronization
- Run the agent whenever you add or modify API endpoints
- Include in your PR checklist: "OpenAPI spec updated"
- Version bump in `openapi.yml` for tracking changes

### 2. Validation Steps
- Always validate the exported JSON against your implementation
- Test that frontend can successfully use the generated specification
- Ensure examples in the spec match actual API responses

### 3. Agent Instructions

When working with the agent, provide clear context:

```
"Update openapi.yml for the new /api/v1/orders endpoints. 
The implementation is in internal/api/controllers/order_controller.go 
with DTOs in internal/api/dto/request/order_request.go"
```

### 4. Frontend Integration

The exported `falcon-openapi.json` should be:
- Placed in your frontend's API client directory
- Used for generating TypeScript interfaces
- Referenced for API client configuration

## Common Scenarios

### Adding New Endpoints
1. Implement the endpoint in your Golang code
2. Ask the agent: "Add the new [endpoint] to openapi.yml"
3. Run export command
4. Update frontend to use new endpoints

### Updating Existing Endpoints
1. Modify your Golang implementation
2. Ask the agent: "Update the [endpoint] schema in openapi.yml to match the new implementation"
3. Run export command
4. Update frontend types/interfaces

### Fixing Discrepancies
1. Ask the agent: "Find and fix discrepancies between internal/api and openapi.yml"
2. Review the proposed changes
3. Run export command
4. Test frontend against updated spec

## Troubleshooting

### Common Issues

1. **Missing Endpoints in Export**
   - Ensure the agent has analyzed all controllers
   - Check route registration in your Golang code
   - Verify path patterns match between code and spec

2. **Schema Mismatches**
   - Ask agent to specifically analyze DTO structs
   - Check JSON tags and binding tags alignment
   - Ensure nullable fields are properly documented

3. **Version Conflicts**
   - Always bump version in openapi.yml after changes
   - Keep track of API version in both spec and code
   - Document breaking changes clearly

## Maintenance Checklist

- [ ] All endpoints in `internal/api` are documented in `openapi.yml`
- [ ] Request/response schemas match DTO definitions
- [ ] Authentication requirements are properly specified
- [ ] Error responses are comprehensively documented
- [ ] Examples are valid and helpful
- [ ] Version number is updated after changes
- [ ] `falcon-openapi.json` is regenerated after updates
- [ ] Frontend can successfully consume the exported specification

## Example Agent Queries

```
# Full synchronization
"Perform a complete synchronization between internal/api and openapi.yml"

# Specific endpoint update
"Update the POST /api/v1/users endpoint documentation to include the new email verification field"

# Schema validation
"Verify that all response DTOs in internal/api/dto/response are properly documented in openapi.yml"

# Security documentation
"Document all authenticated endpoints and their required permissions in openapi.yml"

# Error handling
"Ensure all possible error responses from the API are documented with proper schemas"
```

## Notes

- The agent preserves existing valid documentation while adding missing elements
- It follows OpenAPI 3.0+ specification standards
- The focus is on accuracy and completeness for frontend consumption
- Regular synchronization prevents documentation drift