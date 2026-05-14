package export

import (
	"strings"

	"httpclient/internal/model"
)

type CurlExporter struct{}

func (e *CurlExporter) Format() string { return "curl" }

func (e *CurlExporter) Export(session *model.Session, tree *model.SessionTree) ([]byte, error) {
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
		baseURL := tree.GetInheritedBaseURL(session.ID)
		if baseURL != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(url, "/")
		}
		output.WriteString(" '")
		output.WriteString(url)
		output.WriteString("'")

		// Auth from session
		if session.Auth != nil {
			switch session.Auth.Type {
			case "basic":
				output.WriteString(" \\\n  -u '")
				output.WriteString(session.Auth.Username)
				output.WriteString(":")
				output.WriteString(session.Auth.Password)
				output.WriteString("'")
			case "token":
				tokenType := session.Auth.TokenType
				if tokenType == "" {
					tokenType = "Bearer"
				}
				headerName := session.Auth.HeaderName
				if headerName == "" {
					headerName = "Authorization"
				}
				output.WriteString(" \\\n  -H '")
				output.WriteString(headerName)
				output.WriteString(": ")
				output.WriteString(tokenType)
				output.WriteString(" ")
				output.WriteString(session.Auth.Token)
				output.WriteString("'")
			case "oauth":
				if session.Auth.AccessToken != "" {
					output.WriteString(" \\\n  -H 'Authorization: Bearer ")
					output.WriteString(session.Auth.AccessToken)
					output.WriteString("'")
				}
			}
		}

		// Headers from session
		headers := tree.GetEffectiveHeaders(session.ID)
		for k, v := range headers {
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
