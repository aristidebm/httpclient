package input

import (
	"fmt"
	"strings"

	"httpclient/internal/model"
)

func ParseHTTPFile(data []byte) ([]*model.Request, error) {
	content := string(data)
	blocks := splitHTTPFileBlocks(content)

	var requests []*model.Request
	vars := make(map[string]any)

	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}

		req, blockVars, err := parseHTTPBlock(block)
		if err != nil {
			return nil, err
		}

		// Merge block vars into global vars
		for k, v := range blockVars {
			vars[k] = v
		}

		// Apply variables to request
		if req.URL != "" {
			req.URL, _ = model.ResolveVars(req.URL, vars)
		}
		if len(req.Body) > 0 {
			resolved, _ := model.ResolveVars(string(req.Body), vars)
			req.Body = []byte(resolved)
		}

		requests = append(requests, req)
	}

	return requests, nil
}

func splitHTTPFileBlocks(content string) []string {
	// Split on ### separator
	var blocks []string
	parts := strings.Split(content, "###")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			blocks = append(blocks, part)
		}
	}
	return blocks
}

func parseHTTPBlock(block string) (*model.Request, map[string]any, error) {
	lines := splitLines(block)
	req := &model.Request{
		Headers:     make(map[string]string),
		Params:      make(map[string]string),
		ContentType: "application/json",
	}
	vars := make(map[string]any)

	for i, line := range lines {
		line = trimComment(line)
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// First non-comment line should be METHOD URL
		if i == 0 {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				return nil, nil, fmt.Errorf("invalid request: %s", line)
			}
			req.Method = parts[0]
			req.URL = parts[1]
			continue
		}

		// Check for variable declarations @var=value or @var = value
		if strings.HasPrefix(line, "@") {
			// Remove @ and parse var = value
			line = line[1:]
			// Support both @VAR=value and @VAR = value
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				varName := strings.TrimSpace(parts[0])
				varValue := strings.TrimSpace(parts[1])
				if varName != "" {
					vars[varName] = varValue
				}
			}
			continue
		}

		// Check for header: Key: Value
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				req.Headers[key] = value
			}
			continue
		}

		// Otherwise it's the body
		if req.Body == nil {
			req.Body = []byte(line)
		} else {
			req.Body = append(req.Body, []byte("\n"+line)...)
		}
	}

	// Resolve {{variable}} in URL and headers
	if req.URL != "" {
		req.URL = resolveBraceVars(req.URL, vars)
	}
	for k, v := range req.Headers {
		req.Headers[k] = resolveBraceVars(v, vars)
	}
	if len(req.Body) > 0 {
		resolved := resolveBraceVars(string(req.Body), vars)
		req.Body = []byte(resolved)
	}

	return req, vars, nil
}

func resolveBraceVars(s string, vars map[string]any) string {
	for k, v := range vars {
		placeholder := fmt.Sprintf("{{%s}}", k)
		s = strings.ReplaceAll(s, placeholder, fmt.Sprintf("%v", v))
	}
	return s
}
