package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fujiwara/ridge"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

type Latency struct {
	duration  time.Duration
	randomize bool
}

func (l *Latency) Sleep() {
	if l.duration == 0 {
		return
	}
	var s time.Duration
	if l.randomize {
		s = time.Duration(rand.NormFloat64() * float64(l.duration))
	} else {
		s = l.duration
	}
	time.Sleep(s)
}

var commonLatency = &Latency{}

type OtelConfig struct {
	endpoint    string
	serviceName string
}

// parseOTLPEndpoint parses an OTLP endpoint string that may be a URL (http://host:port)
// or just host:port. Returns the host:port and whether the connection should use TLS.
func parseOTLPEndpoint(rawEndpoint string) (endpoint string, useTLS bool, err error) {
	if strings.Contains(rawEndpoint, "://") {
		u, err := url.Parse(rawEndpoint)
		if err != nil {
			return "", false, fmt.Errorf("failed to parse OTLP endpoint URL: %w", err)
		}
		return u.Host, u.Scheme == "https", nil
	}
	return rawEndpoint, false, nil
}

type accessLogResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (w *accessLogResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *accessLogResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

func accessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &accessLogResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		elapsed := time.Since(start)

		attrs := []slog.Attr{
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("request", fmt.Sprintf("%s %s %s", r.Method, r.URL.RequestURI(), r.Proto)),
			slog.Int("status_code", rw.statusCode),
			slog.Int("body_bytes_sent", rw.bytesWritten),
			slog.String("user_agent", r.UserAgent()),
			slog.String("referer", r.Referer()),
			slog.Int64("response_time_ms", elapsed.Milliseconds()),
		}
		sc := trace.SpanContextFromContext(r.Context())
		if sc.HasTraceID() {
			attrs = append(attrs, slog.String("trace_id", sc.TraceID().String()))
		}
		slog.LogAttrs(r.Context(), slog.LevelInfo, "access", attrs...)
	})
}

func setupTracerProvider(ctx context.Context, config *OtelConfig) (func(context.Context) error, error) {
	if config.endpoint == "" {
		slog.Info("OpenTelemetry tracing disabled", "reason", "no endpoint configured")
		return func(context.Context) error { return nil }, nil
	}

	slog.Info("Setting up OpenTelemetry tracing", "endpoint", config.endpoint, "service", config.serviceName)

	endpoint, useTLS, err := parseOTLPEndpoint(config.endpoint)
	if err != nil {
		return nil, err
	}

	// Create OTLP HTTP exporter
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
	}
	if !useTLS {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Create resource with service name
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(config.serviceName)),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	slog.Info("OpenTelemetry tracing enabled")

	return tp.Shutdown, nil
}

func main() {
	var port int
	var otelConfig OtelConfig

	// Set up JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	flag.IntVar(&port, "port", 8080, "port number")
	flag.DurationVar(&commonLatency.duration, "latency", 0, "average latency")
	flag.BoolVar(&commonLatency.randomize, "randomize", false, "randomize latency")
	flag.StringVar(&otelConfig.endpoint, "otel-endpoint", "", "OpenTelemetry OTLP endpoint (e.g., localhost:4318)")
	flag.StringVar(&otelConfig.serviceName, "otel-service-name", "printenv", "OpenTelemetry service name")
	flag.VisitAll(func(f *flag.Flag) {
		if s := os.Getenv(strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))); s != "" {
			f.Value.Set(s)
		}
	})
	flag.Parse()

	// Auto-detect OTEL configuration from standard environment variables
	if otelConfig.endpoint == "" {
		if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
			otelConfig.endpoint = endpoint
		}
	}
	if otelConfig.serviceName == "printenv" {
		if serviceName := os.Getenv("OTEL_SERVICE_NAME"); serviceName != "" {
			otelConfig.serviceName = serviceName
		}
	}

	slog.Info("server starting",
		"port", port,
		"otel_enabled", otelConfig.endpoint != "",
		"otel_endpoint", otelConfig.endpoint,
		"otel_service", otelConfig.serviceName,
	)
	slog.Info("latency configuration", "avg", commonLatency.duration, "randomize", commonLatency.randomize)

	// Setup OpenTelemetry tracing
	ctx := context.Background()
	shutdown, err := setupTracerProvider(ctx, &otelConfig)
	if err != nil {
		slog.Error("failed to setup tracer provider", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			slog.Error("failed to shutdown tracer provider", "error", err)
		}
	}()

	var mux = http.NewServeMux()
	mux.HandleFunc("/", handlePrintenv)
	mux.HandleFunc("/headers", handleHeaders)

	// Wrap handler with access log and otelhttp middleware for tracing
	handler := accessLogMiddleware(mux)
	if otelConfig.endpoint != "" {
		handler = otelhttp.NewHandler(handler, "printenv",
			otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		)
	}

	ridge.Run(
		fmt.Sprintf(":%d", port),
		"/",
		handler,
	)
}

func newLatencyFromRequest(r *http.Request) (*Latency, error) {
	s := r.URL.Query().Get("latency")
	if s == "" {
		return commonLatency, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return nil, fmt.Errorf("invalid latency: %s: %w", s, err)
	}
	return &Latency{
		duration:  d,
		randomize: commonLatency.randomize,
	}, nil
}

func handlePrintenv(w http.ResponseWriter, r *http.Request) {
	if l, err := newLatencyFromRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	} else {
		l.Sleep()
	}
	ac := r.Header.Get("Accept")
	if strings.Contains(ac, "application/json") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		envs := make(map[string]string, len(os.Environ()))
		for _, v := range os.Environ() {
			kv := strings.SplitN(v, "=", 2)
			envs[kv[0]] = kv[1]
		}
		json.NewEncoder(w).Encode(envs)
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		envs := os.Environ()
		sort.SliceStable(envs, func(i, j int) bool {
			return envs[i] < envs[j]
		})
		for _, v := range envs {
			fmt.Fprintln(w, v)
		}
	}
}

func handleHeaders(w http.ResponseWriter, r *http.Request) {
	if l, err := newLatencyFromRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	} else {
		l.Sleep()
	}
	headers := make(map[string]string, len(r.Header))
	keys := make([]string, 0, len(r.Header))
	for k, v := range r.Header {
		headers[k] = strings.Join(v, ",")
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ac := r.Header.Get("Accept")
	if strings.Contains(ac, "application/json") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(headers)
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		for _, k := range keys {
			fmt.Fprintf(w, "%s: %s\n", k, headers[k])
		}
	}
}
