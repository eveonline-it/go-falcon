# Examples

This directory contains example code demonstrating how to interact with the API.

## Basic Client

The `basic-client` example shows how to make HTTP requests to the gateway API endpoints.

### Running the Example

1. Start the gateway application:
   ```bash
   go run ./cmd/gateway
   ```

2. In another terminal, run the example client:
   ```bash
   go run ./examples/basic-client
   ```

### Expected Output

```
Health Check Response: map[architecture:gateway status:healthy]
Auth Status Response: map[module:auth status:running version:1.0.0]
```