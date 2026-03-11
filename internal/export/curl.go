package export

import (
	"strings"

	"cdapi/internal/model"
)

type CurlExporter struct{}

func (e *CurlExporter) Format() string { return "curl" }

func (e *CurlExporter) Export(session *model.Session, env *model.Environment) ([]byte, error) {
	var output strings.Builder

	for _, req := range session.Requests {
		if !req.IsExecuted() {
			continue
		}

		if output.Len() > 0 {
			output.WriteString("\n### ")
			output.WriteString(req.ID)
			if req.Note != "" {
				output.WriteString(" — ")
				output.WriteString(req.Note)
			}
			output.WriteString("\n")
		}

		output.WriteString("curl")

		// Method
		if req.Method != "GET" {
			output.WriteString(" -X ")
			output.WriteString(req.Method)
		}

		// URL - expand base URL
		url := req.URL
		if env != nil && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = strings.TrimRight(env.BaseURL, "/") + "/" + strings.TrimLeft(url, "/")
		}
		output.WriteString(" '")
		output.WriteString(url)
		output.WriteString("'")

		// Headers
		if env != nil {
			for k, v := range env.Headers {
				output.WriteString(" \\\n  -H '")
				output.WriteString(k)
				output.WriteString(": ")
				output.WriteString(v)
				output.WriteString("'")
			}
		}
		for k, v := range req.Headers {
			output.WriteString(" \\\n  -H '")
			output.WriteString(k)
			output.WriteString(": ")
			output.WriteString(v)
			output.WriteString("'")
		}

		// Body
		if len(req.Body) > 0 {
			output.WriteString(" \\\n  -d '")
			output.WriteString(string(req.Body))
			output.WriteString("'")
		}

		output.WriteString("\n")
	}

	return []byte(output.String()), nil
}

func init() {
	Register("curl", func() Exporter { return &CurlExporter{} })
}
