package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"httpclient/internal/model"
)

type BrunoExporter struct{}

func (e *BrunoExporter) Format() string { return "bruno" }

func (e *BrunoExporter) Export(session *model.Session, env *model.Environment) ([]byte, error) {
	// For now, just print what would be exported
	// Full directory export would need to be handled by the command
	var output strings.Builder

	output.WriteString("# Bruno Export\n\n")
	output.WriteString("Session: ")
	output.WriteString(session.Name)
	output.WriteString("\n")

	if env != nil {
		output.WriteString("Environment: ")
		output.WriteString(env.Name)
		output.WriteString(" (")
		output.WriteString(env.BaseURL)
		output.WriteString(")\n")
	}

	output.WriteString("\nRequests:\n")
	for _, req := range session.Requests {
		output.WriteString(fmt.Sprintf("  - [%s] %s %s", req.ID, req.Method, req.URL))
		if req.Note != "" {
			output.WriteString(" — ")
			output.WriteString(req.Note)
		}
		output.WriteString("\n")
	}

	return []byte(output.String()), nil
}

func ExportBrunoDir(session *model.Session, env *model.Environment, outDir string) error {
	// Create directory
	sessionDir := filepath.Join(outDir, session.Name)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return err
	}

	// Create bruno.json collection file
	collection := map[string]any{
		"name":      session.Name,
		"version":   "1",
		"scripts":   map[string]any{},
		"requests":  []map[string]any{},
		"variables": map[string]any{},
	}

	collectionJSON, _ := json.MarshalIndent(collection, "", "  ")
	os.WriteFile(filepath.Join(sessionDir, "bruno.json"), collectionJSON, 0644)

	// Create environments directory
	envDir := filepath.Join(sessionDir, "environments")
	os.MkdirAll(envDir, 0755)

	if env != nil {
		// Create environment file
		var envContent strings.Builder
		if env.BaseURL != "" {
			fmt.Fprintf(&envContent, "vars {\n  baseUrl: %s\n}\n", env.BaseURL)
		}
		for k, v := range env.Vars {
			fmt.Fprintf(&envContent, "vars {\n  %s: %v\n}\n", k, v)
		}
		os.WriteFile(filepath.Join(envDir, env.Name+".bru"), []byte(envContent.String()), 0644)
	}

	// Create request files
	for _, req := range session.Requests {
		filename := req.ID
		if req.Note != "" {
			filename = req.ID + "-" + strings.ReplaceAll(req.Note, " ", "-")
		}

		var content strings.Builder
		metaName := fmt.Sprintf("%s %s %s", req.ID, req.Method, req.URL)
		if req.Note != "" {
			metaName = req.Note
		}

		fmt.Fprintf(&content, "meta {\n  name: %s\n  type: http\n  seq: %d\n}\n\n", metaName, 1)

		url := req.URL
		if env != nil && !strings.HasPrefix(url, "http") {
			url = "{{baseUrl}}/" + strings.TrimLeft(url, "/")
		}

		fmt.Fprintf(&content, "http {\n  method: %s\n  url: %s\n}\n\n", req.Method, url)

		if len(req.Headers) > 0 {
			content.WriteString("headers {\n")
			for k, v := range req.Headers {
				fmt.Fprintf(&content, "  %s: %s\n", k, v)
			}
			content.WriteString("}\n\n")
		}

		if len(req.Body) > 0 {
			content.WriteString("body:json {\n")
			content.WriteString(string(req.Body))
			content.WriteString("\n}\n")
		}

		os.WriteFile(filepath.Join(sessionDir, filename+".bru"), []byte(content.String()), 0644)
	}

	return nil
}

func init() {
	Register("bruno", func() Exporter { return &BrunoExporter{} })
}
