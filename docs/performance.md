# Performance Guide

## CPU Configuration

The application uses [uber-go/automaxprocs](https://github.com/uber-go/automaxprocs) to automatically configure `GOMAXPROCS` based on container CPU limits.

### What automaxprocs Does

1. **Detects Container Limits**: Reads cgroup CPU limits in containerized environments
2. **Adjusts GOMAXPROCS**: Sets the optimal number of OS threads for Go runtime
3. **Prevents Over-subscription**: Avoids using more threads than allocated CPU quota

### Startup Information

When the application starts, you'll see:
```
🖥️  CPU Configuration:
   - System CPUs: 6
   - GOMAXPROCS: 6
   - automaxprocs: Automatically adjusting based on container limits
maxprocs: Leaving GOMAXPROCS=6: CPU quota undefined
```

### Container Scenarios

#### Docker with CPU Limits
```bash
# Limit to 2 CPUs
docker run --cpus="2" your-app

# automaxprocs will set GOMAXPROCS=2 regardless of host CPU count
```

#### Kubernetes with Resource Limits
```yaml
resources:
  limits:
    cpu: "1.5"  # automaxprocs will set GOMAXPROCS=1
  requests:
    cpu: "0.5"
```

#### No Limits (Development)
```
# automaxprocs leaves GOMAXPROCS unchanged
# Uses system CPU count
```

### Benefits

1. **Better Performance**: Optimal thread count for allocated resources
2. **Resource Efficiency**: Prevents CPU thrashing in constrained environments
3. **Container-Aware**: Works automatically in Docker/Kubernetes
4. **Zero Configuration**: Import and forget

### Monitoring

Check runtime configuration:
```go
import "runtime"

fmt.Printf("NumCPU: %d\n", runtime.NumCPU())
fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
```

### Manual Override

If needed, you can still manually set GOMAXPROCS:
```bash
export GOMAXPROCS=4
./gateway
```

Or in code (before automaxprocs import):
```go
runtime.GOMAXPROCS(4)
```