package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestParseOTLPEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		endpoint string
		useTLS   bool
		wantErr  bool
	}{
		{
			name:     "host:port only",
			input:    "localhost:4318",
			endpoint: "localhost:4318",
			useTLS:   false,
		},
		{
			name:     "http URL",
			input:    "http://localhost:4318",
			endpoint: "localhost:4318",
			useTLS:   false,
		},
		{
			name:     "https URL",
			input:    "https://otel-collector.example.com:4318",
			endpoint: "otel-collector.example.com:4318",
			useTLS:   true,
		},
		{
			name:     "http URL without port",
			input:    "http://localhost",
			endpoint: "localhost",
			useTLS:   false,
		},
		{
			name:     "https URL without port",
			input:    "https://otel-collector.example.com",
			endpoint: "otel-collector.example.com",
			useTLS:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint, useTLS, err := parseOTLPEndpoint(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseOTLPEndpoint(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if endpoint != tt.endpoint {
				t.Errorf("parseOTLPEndpoint(%q) endpoint = %q, want %q", tt.input, endpoint, tt.endpoint)
			}
			if useTLS != tt.useTLS {
				t.Errorf("parseOTLPEndpoint(%q) useTLS = %v, want %v", tt.input, useTLS, tt.useTLS)
			}
		})
	}
}

func TestAccessLogMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	})

	t.Run("without trace context", func(t *testing.T) {
		var buf bytes.Buffer
		slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

		srv := httptest.NewServer(accessLogMiddleware(handler))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/test")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()

		var logEntry map[string]any
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("failed to parse log: %v, raw: %s", err, buf.String())
		}
		if logEntry["msg"] != "access" {
			t.Errorf("msg = %v, want access", logEntry["msg"])
		}
		if _, ok := logEntry["trace_id"]; ok {
			t.Error("trace_id should not be present without trace context")
		}
		if sc, ok := logEntry["status_code"]; !ok || sc != float64(200) {
			t.Errorf("status_code = %v, want 200", sc)
		}
	})

	t.Run("with trace context", func(t *testing.T) {
		var buf bytes.Buffer
		slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

		traceID, _ := trace.TraceIDFromHex("0af7651916cd43dd8448eb211c80319c")
		spanID, _ := trace.SpanIDFromHex("b7ad6b7169203331")
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		})

		wrappedHandler := accessLogMiddleware(handler)
		req := httptest.NewRequest("GET", "/test", nil)
		req = req.WithContext(trace.ContextWithSpanContext(context.Background(), sc))
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)

		var logEntry map[string]any
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("failed to parse log: %v, raw: %s", err, buf.String())
		}
		if logEntry["trace_id"] != "0af7651916cd43dd8448eb211c80319c" {
			t.Errorf("trace_id = %v, want 0af7651916cd43dd8448eb211c80319c", logEntry["trace_id"])
		}
	})
}
