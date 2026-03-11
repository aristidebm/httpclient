package session

import (
	"os"
	"path/filepath"
	"testing"

	"cdapi/internal/model"
)

func TestSaveAndLoadTree(t *testing.T) {
	tmpDir := t.TempDir()

	origConfigDir := ConfigDir
	origSessionsFile := SessionsFile
	ConfigDir = tmpDir
	SessionsFile = filepath.Join(tmpDir, "sessions.json")
	defer func() {
		ConfigDir = origConfigDir
		SessionsFile = origSessionsFile
	}()

	tree := model.NewSessionTree()
	req := &model.Request{
		Method:      "GET",
		URL:         "https://example.com",
		Headers:     map[string]string{"Content-Type": "application/json"},
		Params:      map[string]string{"page": "1"},
		Body:        []byte(`{"name":"test"}`),
		ContentType: "application/json",
	}
	tree.Current().AddRequest(req)

	err := SaveTree(tree)
	if err != nil {
		t.Fatalf("SaveTree failed: %v", err)
	}

	loaded, err := LoadTree()
	if err != nil {
		t.Fatalf("LoadTree failed: %v", err)
	}

	if loaded.CurrentID != tree.CurrentID {
		t.Errorf("expected currentID %s, got %s", tree.CurrentID, loaded.CurrentID)
	}

	current := loaded.Current()
	if current == nil {
		t.Fatal("expected current session")
	}
	if len(current.Requests) != 1 {
		t.Errorf("expected 1 request, got %d", len(current.Requests))
	}
}

func TestLoadTreeCorrupt(t *testing.T) {
	tmpDir := t.TempDir()

	origConfigDir := ConfigDir
	origSessionsFile := SessionsFile
	ConfigDir = tmpDir
	SessionsFile = filepath.Join(tmpDir, "sessions.json")
	defer func() {
		ConfigDir = origConfigDir
		SessionsFile = origSessionsFile
	}()

	os.WriteFile(filepath.Join(tmpDir, "sessions.json"), []byte("not valid json{{{"), 0644)

	loaded, err := LoadTree()
	if err != nil {
		t.Fatalf("LoadTree should not fail on corrupt file: %v", err)
	}

	if loaded.CurrentID != "default" {
		t.Errorf("expected default session on corrupt file, got %s", loaded.CurrentID)
	}
}

func TestRequestWithBinaryBody(t *testing.T) {
	tmpDir := t.TempDir()

	origConfigDir := ConfigDir
	origSessionsFile := SessionsFile
	ConfigDir = tmpDir
	SessionsFile = filepath.Join(tmpDir, "sessions.json")
	defer func() {
		ConfigDir = origConfigDir
		SessionsFile = origSessionsFile
	}()

	binaryBody := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}

	tree := model.NewSessionTree()
	req := &model.Request{
		Method:      "POST",
		URL:         "https://example.com/upload",
		Headers:     map[string]string{"Content-Type": "application/octet-stream"},
		Body:        binaryBody,
		ContentType: "application/octet-stream",
	}
	tree.Current().AddRequest(req)

	err := SaveTree(tree)
	if err != nil {
		t.Fatalf("SaveTree failed: %v", err)
	}

	loaded, err := LoadTree()
	if err != nil {
		t.Fatalf("LoadTree failed: %v", err)
	}

	current := loaded.Current()
	if len(current.Requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(current.Requests))
	}

	loadedReq := current.Requests[0]
	if string(loadedReq.Body) != string(binaryBody) {
		t.Errorf("expected body %v, got %v", binaryBody, loadedReq.Body)
	}
}

func TestLoadConfigDefault(t *testing.T) {
	tmpDir := t.TempDir()

	origConfigDir := ConfigDir
	origConfigFile := ConfigFile
	ConfigDir = tmpDir
	ConfigFile = filepath.Join(tmpDir, "config.yaml")
	defer func() {
		ConfigDir = origConfigDir
		ConfigFile = origConfigFile
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.DefaultEnv != "local" {
		t.Errorf("expected default_env 'local', got %s", cfg.DefaultEnv)
	}
	if cfg.HistoryFile == "" {
		t.Error("expected history_file to be set")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()

	origConfigDir := ConfigDir
	origConfigFile := ConfigFile
	ConfigDir = tmpDir
	ConfigFile = filepath.Join(tmpDir, "config.yaml")
	defer func() {
		ConfigDir = origConfigDir
		ConfigFile = origConfigFile
	}()

	cfg := &Config{
		DefaultEnv:    "beta",
		DefaultEditor: "nano",
		HistoryFile:   "~/.cdapi/history",
	}

	err := SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.DefaultEnv != "beta" {
		t.Errorf("expected DefaultEnv 'beta', got %s", loaded.DefaultEnv)
	}
	if loaded.DefaultEditor != "nano" {
		t.Errorf("expected DefaultEditor 'nano', got %s", loaded.DefaultEditor)
	}
}
