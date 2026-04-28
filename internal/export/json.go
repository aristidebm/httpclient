package export

import (
	"encoding/json"

	"httpclient/internal/model"
)

type JSONExporter struct{}

func (e *JSONExporter) Format() string { return "json" }

func (e *JSONExporter) Export(session *model.Session, tree *model.SessionTree) ([]byte, error) {
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

	type Sess struct {
		Name      string `json:"name"`
		BaseURL   string `json:"baseUrl,omitempty"`
		ParentID  string `json:"parentId,omitempty"`
		CreatedAt string `json:"createdAt"`
		Requests  []Req  `json:"requests"`
	}

	// Convert Variables to map[string]any for export
	varsMap := make(map[string]any)
	vars := tree.GetEffectiveVars(session.ID)
	for k, v := range vars {
		varsMap[k] = v.Value
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

	baseURL := tree.GetInheritedBaseURL(session.ID)
	output := map[string]any{
		"session": Sess{
			Name:      session.Name,
			BaseURL:   baseURL,
			ParentID:  session.ParentID,
			CreatedAt: session.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Requests:  requests,
		},
	}

	// Include effective headers and vars
	if len(varsMap) > 0 {
		output["variables"] = varsMap
	}

	headers := tree.GetEffectiveHeaders(session.ID)
	if len(headers) > 0 {
		output["headers"] = headers
	}

	return json.MarshalIndent(output, "", "  ")
}

func init() {
	Register("json", func() Exporter { return &JSONExporter{} })
}
