package export

import (
	"strings"

	"cdapi/internal/model"
)

type HTTPFileExporter struct{}

func (e *HTTPFileExporter) Format() string { return "http" }

func (e *HTTPFileExporter) Export(session *model.Session, env *model.Environment) ([]byte, error) {
	var output strings.Builder

	// Variable declarations at top
	if env != nil {
		output.WriteString("@baseUrl = ")
		output.WriteString(env.BaseURL)
		output.WriteString("\n")
		if auth, ok := env.Headers["Authorization"]; ok {
			output.WriteString("@token = ")
			output.WriteString(auth)
			output.WriteString("\n")
		}
		output.WriteString("\n")
	}

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
		if env != nil && env.BaseURL != "" {
			url = strings.ReplaceAll(url, env.BaseURL, "{{baseUrl}}")
		}

		output.WriteString(req.Method)
		output.WriteString(" ")
		output.WriteString(url)
		output.WriteString("\n")

		// Headers
		for k, v := range req.Headers {
			if env != nil {
				auth, hasAuth := env.Headers["Authorization"]
				if hasAuth && auth != "" {
					v = strings.ReplaceAll(v, auth, "{{token}}")
				}
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
