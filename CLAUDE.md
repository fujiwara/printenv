# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

printenv is a simple HTTP server that prints environment variables and request headers. It can run as a standalone server or on AWS Lambda via API Gateway (using fujiwara/ridge).

The entire application is contained in a single Go file (`main.go`) with two main endpoints:
- `/` - Returns environment variables (text/plain or application/json based on Accept header)
- `/headers` - Returns request headers (text/plain or application/json based on Accept header)

Both endpoints support latency simulation via `-latency` flag, `LATENCY` env var, or `?latency=` query parameter.

## Development Commands

### Build and Test
```bash
# Run tests
go test -race -v ./...

# Build binary
go build .

# Build (with tests)
make build

# Test only
make test
```

### Format Code
```bash
# Format all Go files (REQUIRED before commit)
go fmt ./...
```

### Docker
```bash
# Build Docker image
make image

# Run locally
docker run -p 8080:8080 ghcr.io/fujiwara/printenv:$(git describe --tags)
```

### Local Development
```bash
# Run server locally
go run main.go

# Run with latency simulation
go run main.go -latency 100ms -randomize

# Run with OpenTelemetry tracing
go run main.go -otel-endpoint localhost:4318 -otel-service-name printenv

# Test endpoints
curl localhost:8080
curl -H "Accept: application/json" localhost:8080
curl localhost:8080/headers
```

## Architecture Notes

- **Single-file application**: All code is in `main.go`
- **ridge integration**: Uses fujiwara/ridge to enable both standard HTTP server and AWS Lambda deployment without code changes
- **Content negotiation**: Responses automatically switch between plain text and JSON based on Accept header
- **Latency simulation**: Global latency can be set via CLI/env, overridden per-request via query parameter
- **OpenTelemetry tracing**: HTTP requests are traced using otelhttp middleware when OTLP endpoint is configured
- **Access logging**: Custom slog-based access log middleware outputs JSON access logs with trace_id when tracing is enabled

## OpenTelemetry Tracing

The application supports distributed tracing via OpenTelemetry with OTLP HTTP exporter.

### Configuration

Tracing is automatically enabled when an OTLP endpoint is configured. Configuration can be done via CLI flags or environment variables (flags take precedence):

**CLI Flags:**
- `-otel-endpoint`: OTLP HTTP endpoint (e.g., `localhost:4318`)
- `-otel-service-name`: Service name for traces (default: `printenv`)

**Environment Variables:**
- `OTEL_EXPORTER_OTLP_ENDPOINT`: Standard OpenTelemetry endpoint variable
- `OTEL_SERVICE_NAME`: Standard OpenTelemetry service name variable
- `OTEL_ENDPOINT`: Alternative endpoint variable

### Example with Jaeger

```bash
# Run Jaeger all-in-one
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest

# Run printenv with tracing
go run main.go -otel-endpoint localhost:4318

# Or using environment variables
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
go run main.go

# Access Jaeger UI at http://localhost:16686
```

### Trace Information

Each HTTP request generates a trace with:
- Span name: "printenv"
- HTTP method, path, status code
- Request and response headers (configurable)
- Latency timing

## Key Dependencies

- `github.com/fujiwara/ridge`: Enables dual-mode deployment (HTTP server + AWS Lambda)
- `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`: HTTP tracing middleware
- `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp`: OTLP HTTP trace exporter
- Requires Go 1.25+

## CI/CD

- GitHub Actions runs tests on Go 1.25 and 1.26 on every push/PR
- Uses tagpr for automated releases and CHANGELOG management
- Docker images are published to ghcr.io/fujiwara/printenv
