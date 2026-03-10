package main

import (
	"testing"
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
