package executor

import (
	"net/http"
	"testing"
	"time"

	"httpclient/internal/model"
)

func TestClientExecuteSuccess(t *testing.T) {
	client := NewClient(10 * time.Second)

	env := &model.Environment{
		Name:    "test",
		BaseURL: "https://httpbin.org",
		Headers: map[string]string{},
		Vars:    model.Variables{},
	}

	req := &model.Request{
		Method: "GET",
		URL:    "/get",
		Headers: map[string]string{
			"Accept": "application/json",
		},
		Vars: model.Variables{},
	}

	err := client.Execute(req, env)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if req.Response == nil {
		t.Fatal("expected Response to be set")
	}
	if req.Response.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", req.Response.StatusCode)
	}
	if req.ExecutedAt.IsZero() {
		t.Error("expected ExecutedAt to be set")
	}
	if req.Duration == 0 {
		t.Error("expected Duration to be set")
	}
}

func TestClientTimeout(t *testing.T) {
	client := NewClient(1 * time.Millisecond)

	env := &model.Environment{
		Name:    "test",
		BaseURL: "https://httpbin.org",
		Headers: map[string]string{},
		Vars:    model.Variables{},
	}

	req := &model.Request{
		Method: "GET",
		URL:    "https://httpbin.org/delay/5",
		Vars:   model.Variables{},
	}

	err := client.Execute(req, env)
	if err == nil {
		t.Fatal("expected error for timeout")
	}

	execErr, ok := err.(*ExecutorError)
	if !ok {
		t.Fatalf("expected ExecutorError, got %T", err)
	}
	if execErr.Kind != "timeout" {
		t.Errorf("expected kind 'timeout', got %s", execErr.Kind)
	}
}

func TestHeaderMerging(t *testing.T) {
	client := NewClient(10 * time.Second)

	env := &model.Environment{
		Name:    "test",
		BaseURL: "https://httpbin.org",
		Headers: map[string]string{
			"X-Foo": "env",
		},
		Vars: model.Variables{},
	}

	req := &model.Request{
		Method: "GET",
		URL:    "/headers",
		Headers: map[string]string{
			"X-Foo": "request",
		},
		Vars: model.Variables{},
	}

	err := client.Execute(req, env)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if req.Response.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", req.Response.StatusCode)
	}
}

func TestURLResolution(t *testing.T) {
	client := NewClient(10 * time.Second)

	env := &model.Environment{
		Name:    "test",
		BaseURL: "https://httpbin.org",
		Headers: map[string]string{},
		Vars:    model.Variables{},
	}

	req := &model.Request{
		Method: "GET",
		URL:    "/anything/{path}",
		Vars: model.Variables{
			"path": {Name: "path", Value: "test"},
		},
	}

	err := client.Execute(req, env)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if req.Response.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", req.Response.StatusCode)
	}
}

func TestNon2xxResponse(t *testing.T) {
	client := NewClient(10 * time.Second)

	env := &model.Environment{
		Name:    "test",
		BaseURL: "https://httpbin.org",
		Headers: map[string]string{},
		Vars:    model.Variables{},
	}

	req := &model.Request{
		Method: "GET",
		URL:    "/status/404",
		Vars:   model.Variables{},
	}

	err := client.Execute(req, env)
	if err != nil {
		t.Fatalf("expected no error for non-2xx, got %v", err)
	}

	if req.Response.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", req.Response.StatusCode)
	}
}

func TestApplyAuth(t *testing.T) {
	env := &model.Environment{
		Name:    "test",
		BaseURL: "https://api.example.com",
		Headers: map[string]string{
			"Authorization": "my-token",
		},
		Vars: model.Variables{},
	}

	req, _ := http.NewRequest("GET", "https://api.example.com/test", nil)
	ApplyAuth(req, env)

	auth := req.Header.Get("Authorization")
	if auth != "TOKEN my-token" {
		t.Errorf("expected 'TOKEN my-token', got %s", auth)
	}
}

func TestApplyAuthWithTokenPrefix(t *testing.T) {
	env := &model.Environment{
		Name:    "test",
		BaseURL: "https://api.example.com",
		Headers: map[string]string{
			"Authorization": "TOKEN my-token",
		},
		Vars: model.Variables{},
	}

	req, _ := http.NewRequest("GET", "https://api.example.com/test", nil)
	ApplyAuth(req, env)

	auth := req.Header.Get("Authorization")
	if auth != "TOKEN my-token" {
		t.Errorf("expected 'TOKEN my-token', got %s", auth)
	}
}
