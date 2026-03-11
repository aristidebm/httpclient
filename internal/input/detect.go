package input

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

type InputFormat string

const (
	FormatOpenAPI  InputFormat = "openapi"
	FormatCurl     InputFormat = "curl"
	FormatHTTPFile InputFormat = "httpfile"
	FormatUnknown  InputFormat = "unknown"
)

func Detect(data []byte) (InputFormat, error) {
	// Try JSON parse
	var js map[string]any
	if err := json.Unmarshal(data, &js); err == nil {
		if _, ok := js["openapi"]; ok {
			return FormatOpenAPI, nil
		}
		if _, ok := js["swagger"]; ok {
			return FormatOpenAPI, nil
		}
	}

	// Try YAML parse
	var ym map[string]any
	if err := yaml.Unmarshal(data, &ym); err == nil {
		if _, ok := ym["openapi"]; ok {
			return FormatOpenAPI, nil
		}
		if _, ok := ym["swagger"]; ok {
			return FormatOpenAPI, nil
		}
	}

	// Check for curl commands
	content := string(data)
	if containsLine(content, "curl ") {
		return FormatCurl, nil
	}

	// Check for HTTP file format
	lines := splitLines(content)
	for _, line := range lines {
		line = trimComment(line)
		if line == "" {
			continue
		}
		if isHTTPMethod(line) {
			return FormatHTTPFile, nil
		}
	}

	return FormatUnknown, nil
}

func containsLine(content, prefix string) bool {
	lines := splitLines(content)
	for _, line := range lines {
		if len(line) >= len(prefix) && line[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func splitLines(content string) []string {
	var lines []string
	prev := 0
	for i, c := range content {
		if c == '\n' {
			lines = append(lines, content[prev:i])
			prev = i + 1
		}
	}
	if prev < len(content) {
		lines = append(lines, content[prev:])
	}
	return lines
}

func trimComment(line string) string {
	// Remove comments
	for i, c := range line {
		if c == '#' || (i+1 < len(line) && line[i] == '/' && line[i+1] == '/') {
			return line[:i]
		}
	}
	return line
}

func isHTTPMethod(line string) bool {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "TRACE", "CONNECT"}
	for _, m := range methods {
		if len(line) >= len(m) && line[:len(m)] == m {
			return true
		}
	}
	return false
}
