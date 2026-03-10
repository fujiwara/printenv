# printenv

printenv runs a http server that print environment variables.

```console
$ docker run -p 8080:8080 ghcr.io/fujiwara/printenv:v0.2.3
2021/08/13 08:34:01 starting up with local httpd :8080
```

```console
$ curl -s localhost:8080
HOME=/
HOSTNAME=e48a90ce50f8
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
PORT=8080

$ curl -s -H "accept: application/json" localhost:8080 | jq .
{
  "HOME": "/",
  "HOSTNAME": "a0e3875fed32",
  "PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
  "PORT": "8080"
}
```

## Show request headers

`/headers` shows request headers.

```console
$ curl -s localhost:8080/headers
Accept: */*
User-Agent: curl/7.68.0

$ curl -s -H "accept: application/json" localhost:8080/headers | jq .
{
  "Accept": "*/*"
  "User-Agent": "curl/7.68.0"
}
```

## Latency feature

You can add latency to the response.

Set CLI option `-latency` or environment variable `LATENCY` to add latency.

```console
- `-latency [time.Duration]` adds a latency into the response.
  - `LATENCY` environment variable can be used to set the latency.
- `-randomize` randomize latencies.
  - `RANDOMIZE` environment variable can be used to set the randomize.
```

By individual request, you can override latency by query parameter `latency`, for example: `?latency=1s`.

## OpenTelemetry Tracing

printenv supports distributed tracing via OpenTelemetry with OTLP HTTP exporter.

Tracing is automatically enabled when an OTLP endpoint is configured.

### Configuration

**CLI Flags:**
- `-otel-endpoint` - OTLP HTTP endpoint (e.g., `localhost:4318` or `http://localhost:4318`)
- `-otel-service-name` - Service name for traces (default: `printenv`)

**Environment Variables:**
- `OTEL_EXPORTER_OTLP_ENDPOINT` - Standard OpenTelemetry endpoint variable
- `OTEL_SERVICE_NAME` - Standard OpenTelemetry service name variable
- `OTEL_ENDPOINT` - Alternative endpoint variable

### Example with Jaeger

```console
# Run Jaeger all-in-one
$ docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest

# Run printenv with tracing enabled
$ printenv -otel-endpoint localhost:4318

# Or using environment variables
$ export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
$ printenv

# Access Jaeger UI at http://localhost:16686
```

Each HTTP request generates a trace with span name "printenv", including HTTP method, path, status code, and latency timing.

When tracing is enabled, access logs include `trace_id` field for correlating logs with traces.

## LICENSE

MIT License
