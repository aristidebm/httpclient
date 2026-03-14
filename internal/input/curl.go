package input

import (
	"fmt"
	"regexp"
	"strings"

	"httpclient/internal/model"
)

func ParseCurl(data []byte) ([]*model.Request, error) {
	content := string(data)

	// Handle multiline with backslash continuation
	content = handleBackslashContinuation(content)

	// Split into individual curl commands
	commands := splitCurlCommands(content)

	var requests []*model.Request

	for _, cmd := range commands {
		if strings.TrimSpace(cmd) == "" {
			continue
		}

		req, err := parseSingleCurl(cmd)
		if err != nil {
			return nil, err
		}
		if req != nil {
			requests = append(requests, req)
		}
	}

	return requests, nil
}

func handleBackslashContinuation(content string) string {
	// Replace \<newline> with nothing
	re := regexp.MustCompile(`\\[ \t]*\n[ \t]*`)
	return re.ReplaceAllString(content, " ")
}

func splitCurlCommands(content string) []string {
	// First try splitting by ###
	if strings.Contains(content, "###") {
		var commands []string
		parts := strings.Split(content, "###")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" && strings.Contains(part, "curl") {
				commands = append(commands, part)
			}
		}
		if len(commands) > 0 {
			return commands
		}
	}

	// Otherwise split by lines starting with curl
	var commands []string
	var current strings.Builder
	lines := splitLines(content)
	for _, line := range lines {
		if strings.Contains(line, "curl ") || strings.HasPrefix(strings.TrimSpace(line), "curl ") {
			if current.Len() > 0 {
				commands = append(commands, current.String())
				current.Reset()
			}
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	if current.Len() > 0 {
		commands = append(commands, current.String())
	}

	return commands
}

func parseSingleCurl(cmd string) (*model.Request, error) {
	cmd = strings.TrimSpace(cmd)
	if !strings.Contains(cmd, "curl") {
		return nil, nil
	}

	req := &model.Request{
		Headers:     make(map[string]string),
		Params:      make(map[string]string),
		ContentType: "application/json",
		Vars:        make(model.Variables),
	}

	// Extract URL from --url or last argument
	urlMatch := regexp.MustCompile(`--url\s+(\S+)`).FindStringSubmatch(cmd)
	if urlMatch != nil {
		req.URL = urlMatch[1]
	} else {
		// Try to find URL as positional argument
		fields := strings.Fields(cmd)
		for _, f := range fields {
			// Skip flags and their values
			if strings.HasPrefix(f, "-") {
				continue
			}
			// Skip curl command itself
			if f == "curl" {
				continue
			}
			// First non-flag argument is URL
			req.URL = f
			break
		}
	}

	if req.URL == "" {
		return nil, fmt.Errorf("no URL found in curl command")
	}

	// Method
	if m := regexp.MustCompile(`-X\s+(\S+)`).FindStringSubmatch(cmd); m != nil {
		req.Method = m[1]
	} else if strings.Contains(cmd, "-d ") || strings.Contains(cmd, "--data") {
		req.Method = "POST"
	} else if strings.Contains(cmd, "-G ") || strings.Contains(cmd, "--get") {
		req.Method = "GET"
	} else {
		req.Method = "GET"
	}

	// Headers
	headerRe := regexp.MustCompile(`-H\s+['"]?([^'"]+)['"]?\s*`)
	headers := headerRe.FindAllStringSubmatch(cmd, -1)
	for _, h := range headers {
		if len(h) >= 2 {
			parts := strings.SplitN(h[1], ":", 2)
			if len(parts) == 2 {
				req.Headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Data
	if d := regexp.MustCompile(`-d\s+['"]([^'"]+)['"]`).FindStringSubmatch(cmd); d != nil {
		req.Body = []byte(d[1])
	} else if d := regexp.MustCompile(`--data\s+['"]([^'"]+)['"]`).FindStringSubmatch(cmd); d != nil {
		req.Body = []byte(d[1])
	} else if d := regexp.MustCompile(`--data-raw\s+['"]([^'"]+)['"]`).FindStringSubmatch(cmd); d != nil {
		req.Body = []byte(d[1])
	} else if d := regexp.MustCompile(`--data-binary\s+['"]?([^'"]+)['"]?`).FindStringSubmatch(cmd); d != nil {
		req.Body = []byte(d[1])
	}

	// User auth
	if u := regexp.MustCompile(`-u\s+(\S+)`).FindStringSubmatch(cmd); u != nil {
		req.Headers["Authorization"] = "Basic " + u[1]
	}

	// Content-Type
	if strings.Contains(cmd, "--json") {
		req.Headers["Content-Type"] = "application/json"
	}

	return req, nil
}
