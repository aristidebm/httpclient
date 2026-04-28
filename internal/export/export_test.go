package export

import (
	"testing"

	"httpclient/internal/model"
)

func TestJSONExporter(t *testing.T) {
	tree := &model.SessionTree{
		Sessions: map[string]*model.Session{
			"test": {
				Name:    "test-session",
				BaseURL: "https://api.example.com",
				Headers: map[string]string{"Authorization": "Bearer token"},
				Vars:    model.Variables{"api_key": {Name: "api_key", Value: "secret"}},
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
			},
		},
		CurrentID: "test",
	}

	session := tree.Sessions["test"]

	exp := &JSONExporter{}
	data, err := exp.Export(session, tree)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty export")
	}
}

func TestCurlExporter(t *testing.T) {
	tree := &model.SessionTree{
		Sessions: map[string]*model.Session{
			"test": {
				Name:    "test-session",
				BaseURL: "https://api.example.com",
				Headers: map[string]string{"Authorization": "Bearer token"},
				Vars:    model.Variables{},
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
			},
		},
		CurrentID: "test",
	}

	session := tree.Sessions["test"]

	exp := &CurlExporter{}
	data, err := exp.Export(session, tree)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := string(data)
	if len(output) == 0 {
		t.Error("expected non-empty export")
	}
}

func TestHTTPFileExporter(t *testing.T) {
	tree := &model.SessionTree{
		Sessions: map[string]*model.Session{
			"test": {
				Name:    "test-session",
				BaseURL: "https://api.example.com",
				Headers: map[string]string{"Authorization": "Bearer token"},
				Vars:    model.Variables{},
				Requests: []*model.Request{
					{
						ID:      "r1",
						Method:  "GET",
						URL:     "/users",
						Headers: map[string]string{"Accept": "application/json"},
					},
				},
			},
		},
		CurrentID: "test",
	}

	session := tree.Sessions["test"]

	exp := &HTTPFileExporter{}
	data, err := exp.Export(session, tree)
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
