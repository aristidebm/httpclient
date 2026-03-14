package export

import (
	"encoding/json"

	"httpclient/internal/model"
)

type JSONExporter struct{}

func (e *JSONExporter) Format() string { return "json" }

func (e *JSONExporter) Export(session *model.Session, env *model.Environment) ([]byte, error) {
	type Resp struct {
		StatusCode int               `json:"statusCode"`
		Status     string            `json:"status"`
		Headers    map[string]string `json:"headers,omitempty"`
		Body       string            `json:"body,omitempty"`
	}

	type Req struct {
		ID          string            `json:"id"`
		Method      string            `json:"method"`
		URL         string            `json:"url"`
		Headers     map[string]string `json:"headers,omitempty"`
		Params      map[string]string `json:"params,omitempty"`
		Body        string            `json:"body,omitempty"`
		ContentType string            `json:"contentType,omitempty"`
		Note        string            `json:"note,omitempty"`
		Response    *Resp             `json:"response,omitempty"`
	}

	type Env struct {
		Name    string            `json:"name"`
		BaseURL string            `json:"baseUrl"`
		Headers map[string]string `json:"headers,omitempty"`
		Vars    map[string]any    `json:"vars,omitempty"`
	}

	// Convert Variables to map[string]any for export
	varsMap := make(map[string]any)
	if env.Vars != nil {
		for k, v := range env.Vars {
			varsMap[k] = v.Value
		}
	}

	requests := make([]Req, len(session.Requests))
	for i, r := range session.Requests {
		body := ""
		if len(r.Body) > 0 {
			body = string(r.Body)
		}

		var resp *Resp
		if r.Response != nil {
			respBody := ""
			if len(r.Response.Body) > 0 {
				respBody = string(r.Response.Body)
			}
			resp = &Resp{
				StatusCode: r.Response.StatusCode,
				Status:     r.Response.Status,
				Headers:    r.Response.Headers,
				Body:       respBody,
			}
		}

		requests[i] = Req{
			ID:          r.ID,
			Method:      r.Method,
			URL:         r.URL,
			Headers:     r.Headers,
			Params:      r.Params,
			Body:        body,
			ContentType: r.ContentType,
			Note:        r.Note,
			Response:    resp,
		}
	}

	output := map[string]any{
		"session": map[string]any{
			"name":      session.Name,
			"envName":   session.EnvName,
			"parentId":  session.ParentID,
			"createdAt": session.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"requests":  requests,
		},
	}

	if env != nil {
		output["environment"] = Env{
			Name:    env.Name,
			BaseURL: env.BaseURL,
			Headers: env.Headers,
			Vars:    varsMap,
		}
	}

	return json.MarshalIndent(output, "", "  ")
}

func init() {
	Register("json", func() Exporter { return &JSONExporter{} })
}
