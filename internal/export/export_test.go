package export

import (
	"testing"

	"httpclient/internal/model"
)

func TestJSONExporter(t *testing.T) {
	env := &model.Environment{
		Name:    "test",
		BaseURL: "https://api.example.com",
		Headers: map[string]string{"Authorization": "Bearer token"},
		Vars:    model.Variables{"api_key": {Name: "api_key", Value: "secret"}},
	}

	session := &model.Session{
		Name:    "test-session",
		EnvName: "test",
		Requests: []*model.Request{
			{
				ID:     "r1",
				Method: "GET",
				URL:    "https://api.example.com/users",
				Response: &model.Response{
					StatusCode: 200,
					Status:     "OK",
					Body:       []byte(`{"users":[]}`),
					RawBody:    []byte(`{"users":[]}`),
				},
			},
		},
	}

	exp := &JSONExporter{}
	data, err := exp.Export(session, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty export")
	}
}

func TestCurlExporter(t *testing.T) {
	env := &model.Environment{
		Name:    "test",
		BaseURL: "https://api.example.com",
		Headers: map[string]string{"Authorization": "Bearer token"},
		Vars:    model.Variables{},
	}

	session := &model.Session{
		Name:    "test-session",
		EnvName: "test",
		Requests: []*model.Request{
			{
				ID:      "r1",
				Method:  "GET",
				URL:     "https://api.example.com/users",
				Headers: map[string]string{"Accept": "application/json"},
				Response: &model.Response{
					StatusCode: 200,
					Status:     "OK",
					Body:       []byte(`{"users":[]}`),
					RawBody:    []byte(`{"users":[]}`),
				},
			},
		},
	}

	exp := &CurlExporter{}
	data, err := exp.Export(session, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := string(data)
	if len(output) == 0 {
		t.Error("expected non-empty export")
	}
}

func TestHTTPFileExporter(t *testing.T) {
	env := &model.Environment{
		Name:    "test",
		BaseURL: "https://api.example.com",
		Headers: map[string]string{"Authorization": "Bearer token"},
		Vars:    model.Variables{},
	}

	session := &model.Session{
		Name:    "test-session",
		EnvName: "test",
		Requests: []*model.Request{
			{
				ID:      "r1",
				Method:  "GET",
				URL:     "/users",
				Headers: map[string]string{"Accept": "application/json"},
			},
		},
	}

	exp := &HTTPFileExporter{}
	data, err := exp.Export(session, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := string(data)
	if len(output) == 0 {
		t.Error("expected non-empty export")
	}
}

func TestGetExporter(t *testing.T) {
	tests := []struct {
		format   string
		expected bool
	}{
		{"json", true},
		{"curl", true},
		{"har", true},
		{"http", true},
		{"bruno", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			_, err := Get(tt.format)
			if tt.expected && err != nil {
				t.Errorf("expected exporter for %s, got error: %v", tt.format, err)
			}
			if !tt.expected && err == nil {
				t.Errorf("expected error for unknown format %s", tt.format)
			}
		})
	}
}
