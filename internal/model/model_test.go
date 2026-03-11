package model

import (
	"testing"
)

func TestRequestClone(t *testing.T) {
	original := &Request{
		ID:          "r1",
		Method:      "GET",
		URL:         "https://example.com",
		Headers:     map[string]string{"Content-Type": "application/json"},
		Params:      map[string]string{"page": "1"},
		Body:        []byte(`{"name":"test"}`),
		ContentType: "application/json",
		Note:        "test request",
	}
	original.Response = &Response{StatusCode: 200}

	cloned := original.Clone()

	if cloned.ID != original.ID {
		t.Errorf("expected ID %s, got %s", original.ID, cloned.ID)
	}
	if cloned.Method != original.Method {
		t.Errorf("expected Method %s, got %s", original.Method, cloned.Method)
	}
	if cloned.URL != original.URL {
		t.Errorf("expected URL %s, got %s", original.URL, cloned.URL)
	}
	if cloned.Response != nil {
		t.Error("expected Response to be nil after clone")
	}
	if cloned.ExecutedAt != original.ExecutedAt {
		t.Error("expected ExecutedAt to be zero after clone")
	}
	if cloned.Duration != original.Duration {
		t.Error("expected Duration to be zero after clone")
	}
	if string(cloned.Body) != string(original.Body) {
		t.Errorf("expected Body %s, got %s", string(original.Body), string(cloned.Body))
	}

	cloned.Headers["X-Custom"] = "value"
	if original.Headers["X-Custom"] != "" {
		t.Error("clone should have independent Headers map")
	}
}

func TestRequestIsExecuted(t *testing.T) {
	req := &Request{}
	if req.IsExecuted() {
		t.Error("expected IsExecuted to return false when Response is nil")
	}

	req.Response = &Response{StatusCode: 200}
	if !req.IsExecuted() {
		t.Error("expected IsExecuted to return true when Response is set")
	}
}

func TestEnvironmentClone(t *testing.T) {
	original := &Environment{
		Name:    "test",
		BaseURL: "https://api.example.com/",
		Headers: map[string]string{"Authorization": "Bearer token"},
		Vars:    map[string]any{"api_key": "secret123"},
	}

	cloned := original.Clone()

	if cloned.Name != original.Name {
		t.Errorf("expected Name %s, got %s", original.Name, cloned.Name)
	}
	if cloned.BaseURL != original.BaseURL {
		t.Errorf("expected BaseURL %s, got %s", original.BaseURL, cloned.BaseURL)
	}

	cloned.Vars["new_key"] = "new_value"
	if original.Vars["new_key"] != nil {
		t.Error("clone should have independent Vars map")
	}
}

func TestEnvironmentResolve(t *testing.T) {
	env := &Environment{
		Name:    "test",
		BaseURL: "https://api.example.com/",
		Headers: map[string]string{"X-Custom": "header-value"},
		Vars:    map[string]any{"var1": "value1"},
	}

	val, ok := env.Resolve("var1")
	if !ok {
		t.Error("expected to resolve var1")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}

	val, ok = env.Resolve("X-Custom")
	if !ok {
		t.Error("expected to resolve X-Custom header")
	}
	if val != "header-value" {
		t.Errorf("expected header-value, got %v", val)
	}

	_, ok = env.Resolve("nonexistent")
	if ok {
		t.Error("expected false for nonexistent key")
	}
}

func TestSessionNextRequestID(t *testing.T) {
	s := &Session{}
	if s.NextRequestID() != "r1" {
		t.Error("expected first ID to be r1")
	}
	s.Requests = append(s.Requests, &Request{ID: "r1"})
	if s.NextRequestID() != "r2" {
		t.Error("expected second ID to be r2")
	}
}

func TestSessionGetRequest(t *testing.T) {
	s := &Session{
		Requests: []*Request{
			{ID: "r1", Method: "GET"},
			{ID: "r2", Method: "POST"},
		},
	}

	req, ok := s.GetRequest("r1")
	if !ok {
		t.Error("expected to find r1")
	}
	if req.Method != "GET" {
		t.Errorf("expected GET, got %s", req.Method)
	}

	_, ok = s.GetRequest("r3")
	if ok {
		t.Error("expected not found for r3")
	}
}

func TestSessionAddRequest(t *testing.T) {
	s := &Session{}
	req := &Request{Method: "GET", URL: "https://example.com"}

	s.AddRequest(req)

	if req.ID != "r1" {
		t.Errorf("expected ID r1, got %s", req.ID)
	}
	if len(s.Requests) != 1 {
		t.Error("expected 1 request in session")
	}
}

func TestSessionRemoveRequest(t *testing.T) {
	s := &Session{
		Requests: []*Request{
			{ID: "r1"},
			{ID: "r2"},
			{ID: "r3"},
		},
	}

	if !s.RemoveRequest("r2") {
		t.Error("expected remove to succeed")
	}
	if len(s.Requests) != 2 {
		t.Error("expected 2 requests after removal")
	}
	if s.Requests[1].ID != "r3" {
		t.Errorf("expected remaining request to be r3, got %s", s.Requests[1].ID)
	}

	if s.RemoveRequest("r4") {
		t.Error("expected remove of nonexistent to fail")
	}
}

func TestNewSessionTree(t *testing.T) {
	tree := NewSessionTree()

	if tree.CurrentID != "default" {
		t.Errorf("expected CurrentID default, got %s", tree.CurrentID)
	}

	current := tree.Current()
	if current == nil {
		t.Error("expected current session to not be nil")
	}
	if current.Name != "default" {
		t.Errorf("expected session name default, got %s", current.Name)
	}

	env := tree.CurrentEnv()
	if env == nil {
		t.Error("expected current env to not be nil")
	}
	if env.Name != "local" {
		t.Errorf("expected env name local, got %s", env.Name)
	}
}

func TestSessionTreeCurrentEnv(t *testing.T) {
	tree := NewSessionTree()
	env := tree.CurrentEnv()
	if env == nil {
		t.Fatal("expected env")
	}
	env.SetBaseURL("https://api.example.com")
	if env.BaseURL != "https://api.example.com/" {
		t.Errorf("expected trailing slash, got %s", env.BaseURL)
	}
}

func TestSessionTreeChildren(t *testing.T) {
	tree := NewSessionTree()

	tree.Sessions["child"] = &Session{
		ID:       "child",
		Name:     "child",
		ParentID: "default",
	}

	children := tree.Children("default")
	if len(children) != 1 {
		t.Errorf("expected 1 child, got %d", len(children))
	}
	if children[0].ID != "child" {
		t.Errorf("expected child id child, got %s", children[0].ID)
	}
}

func TestResolveVars(t *testing.T) {
	layers := []map[string]any{
		{"request_var": "from_request"},
		{"session_var": "from_session"},
		{"env_var": "from_env"},
		{"global_var": "from_global"},
	}

	tests := []struct {
		template   string
		expected   string
		unresolved int
	}{
		{"{env_var}", "from_env", 0},
		{"{request_var}", "from_request", 0},
		{"prefix_{env_var}_suffix", "prefix_from_env_suffix", 0},
		{"{nonexistent}", "{nonexistent}", 1},
		{"{env_var}_{nonexistent}", "from_env_{nonexistent}", 1},
		{"no_vars_here", "no_vars_here", 0},
		{"{global_var}", "from_global", 0},
		{"{request_var}/{env_var}", "from_request/from_env", 0},
	}

	for _, tt := range tests {
		result, unresolved := ResolveVars(tt.template, layers...)
		if result != tt.expected {
			t.Errorf("ResolveVars(%q) = %q, want %q", tt.template, result, tt.expected)
		}
		if len(unresolved) != tt.unresolved {
			t.Errorf("ResolveVars(%q) unresolved = %d, want %d", tt.template, len(unresolved), tt.unresolved)
		}
	}
}

func TestResolveVarsPriorityOrder(t *testing.T) {
	layers := []map[string]any{
		{"key": "request"},
		{"key": "session"},
		{"key": "env"},
		{"key": "global"},
	}

	result, _ := ResolveVars("{key}", layers...)
	if result != "request" {
		t.Errorf("expected priority to request, got %s", result)
	}

	result, _ = ResolveVars("{key}", layers[1:]...)
	if result != "session" {
		t.Errorf("expected priority to session, got %s", result)
	}
}
