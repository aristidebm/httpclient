package export

import (
	"strings"

	"httpclient/internal/model"
)

type HTTPFileExporter struct{}

func (e *HTTPFileExporter) Format() string { return "http" }

func (e *HTTPFileExporter) Export(session *model.Session, tree *model.SessionTree) ([]byte, error) {
	var output strings.Builder

	// Variable declarations at top
	baseURL := tree.GetInheritedBaseURL(session.ID)
	if baseURL != "" {
		output.WriteString("@baseUrl = ")
		output.WriteString(baseURL)
		output.WriteString("\n")
	}

	auth := tree.GetInheritedAuth(session.ID)
	if auth != nil && auth.Type == "token" {
		output.WriteString("@token = ")
		output.WriteString(auth.TokenType + " " + auth.Token)
		output.WriteString("\n")
	}
	output.WriteString("\n")

	for i, req := range session.Requests {
		if i > 0 {
			output.WriteString("\n### ")
			output.WriteString(req.ID)
			if req.Note != "" {
				output.WriteString(" — ")
				output.WriteString(req.Note)
			}
			output.WriteString("\n")
		}

		// URL with variable substitution
		url := req.URL
		if baseURL != "" {
			url = strings.ReplaceAll(url, baseURL, "{{baseUrl}}")
		}

		output.WriteString(req.Method)
		output.WriteString(" ")
		output.WriteString(url)
		output.WriteString("\n")

		// Headers
		for k, v := range req.Headers {
			if auth != nil && auth.Type == "token" {
				tokenStr := auth.TokenType + " " + auth.Token
				v = strings.ReplaceAll(v, tokenStr, "{{token}}")
			}
			output.WriteString(k)
			output.WriteString(": ")
			output.WriteString(v)
			output.WriteString("\n")
		}

		// Body
		if len(req.Body) > 0 {
			output.WriteString("\n")
			output.WriteString(string(req.Body))
		}

		output.WriteString("\n")
	}

	return []byte(output.String()), nil
}

func init() {
	Register("http", func() Exporter { return &HTTPFileExporter{} })
}
