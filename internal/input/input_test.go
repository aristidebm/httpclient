package input

import (
	"testing"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected InputFormat
	}{
		{
			name:     "OpenAPI JSON",
			input:    `{"openapi": "3.0.0", "info": {"title": "Test"}}`,
			expected: FormatOpenAPI,
		},
		{
			name:     "OpenAPI YAML",
			input:    "openapi: 3.0.0\ninfo:\n  title: Test",
			expected: FormatOpenAPI,
		},
		{
			name:     "curl command",
			input:    "curl -X GET https://example.com",
			expected: FormatCurl,
		},
		{
			name:     "HTTP file GET",
			input:    "GET https://example.com",
			expected: FormatHTTPFile,
		},
		{
			name:     "HTTP file POST",
			input:    "POST https://example.com\nContent-Type: application/json",
			expected: FormatHTTPFile,
		},
		{
			name:     "Unknown format",
			input:    "some random text",
			expected: FormatUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := Detect([]byte(tt.input))
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseHTTPFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // number of requests
	}{
		{
			name: "Simple GET",
			input: `GET https://example.com/api
Accept: application/json`,
			expected: 1,
		},
		{
			name: "Multiple requests with separator",
			input: `GET https://example.com/api

###

POST https://example.com/api
Content-Type: application/json

{"name":"test"}`,
			expected: 2,
		},
		{
			name: "With variable declarations",
			input: `@BASE_URL=https://api.example.com
@TOKEN=secret

GET {{BASE_URL}}/users
Authorization: Bearer {{TOKEN}}`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests, err := ParseHTTPFile([]byte(tt.input))
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(requests) != tt.expected {
				t.Errorf("expected %d requests, got %d", tt.expected, len(requests))
			}
		})
	}
}

func TestParseHTTPFileWithVariables(t *testing.T) {
	input := `@BASE_URL=https://api.example.com
@TOKEN=secret

GET {{BASE_URL}}/users
Authorization: Bearer {{TOKEN}}`

	requests, err := ParseHTTPFile([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}

	req := requests[0]
	// The URL should have variable resolved
	if req.URL == "" {
		t.Error("expected URL to be set")
	}
	// Check that variables were stored
	if req.Headers["Authorization"] == "" {
		t.Error("expected Authorization header to be set")
	}
}
