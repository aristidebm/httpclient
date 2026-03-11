package export

import (
	"encoding/json"
	"time"

	"httpclient/internal/model"
)

type HARExporter struct{}

func (e *HARExporter) Format() string { return "har" }

func (e *HARExporter) Export(session *model.Session, env *model.Environment) ([]byte, error) {
	entries := make([]map[string]any, 0)

	for _, req := range session.Requests {
		if !req.IsExecuted() {
			continue
		}

		// Distribute duration evenly
		timing := map[string]any{
			"send":    req.Duration.Milliseconds() / 3,
			"wait":    req.Duration.Milliseconds() / 3,
			"receive": req.Duration.Milliseconds() / 3,
		}

		headers := []map[string]any{}
		for k, v := range req.Response.Headers {
			headers = append(headers, map[string]any{"name": k, "value": v})
		}

		reqHeaders := []map[string]any{}
		for k, v := range req.Headers {
			reqHeaders = append(reqHeaders, map[string]any{"name": k, "value": v})
		}

		entry := map[string]any{
			"startedDateTime": req.ExecutedAt.Format(time.RFC3339),
			"time":            req.Duration.Milliseconds(),
			"request": map[string]any{
				"method":      req.Method,
				"url":         req.URL,
				"headers":     reqHeaders,
				"queryString": []map[string]any{},
				"postData": map[string]any{
					"mimeType": req.ContentType,
					"text":     string(req.Body),
				},
			},
			"response": map[string]any{
				"status":     req.Response.StatusCode,
				"statusText": req.Response.Status,
				"headers":    headers,
				"content": map[string]any{
					"size":     len(req.Response.RawBody),
					"mimeType": req.Response.Headers["Content-Type"],
					"text":     string(req.Response.RawBody),
				},
			},
			"timings": timing,
		}

		entries = append(entries, entry)
	}

	har := map[string]any{
		"log": map[string]any{
			"version": "1.2",
			"creator": map[string]any{
				"name":    "httpclient",
				"version": "1.0",
			},
			"entries": entries,
		},
	}

	return json.MarshalIndent(har, "", "  ")
}

func init() {
	Register("har", func() Exporter { return &HARExporter{} })
}
