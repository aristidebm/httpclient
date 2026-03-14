package executor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"httpclient/internal/model"
)

var ErrTimeout = errors.New("timeout")

type ExecutorError struct {
	Kind  string
	Cause error
}

func (e *ExecutorError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Kind, e.Cause)
	}
	return e.Kind
}

func (e *ExecutorError) Unwrap() error {
	return e.Cause
}

type Client struct {
	httpClient *http.Client
}

func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Execute(req *model.Request, env *model.Environment) error {
	mergedHeaders := make(map[string]string)
	for k, v := range env.Headers {
		mergedHeaders[k] = v
	}
	for k, v := range req.Headers {
		mergedHeaders[k] = v
	}

	fullURL := req.URL
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		if env.BaseURL != "" {
			baseURL := strings.TrimRight(env.BaseURL, "/")
			if strings.HasPrefix(req.URL, "/") {
				fullURL = baseURL + req.URL
			} else {
				fullURL = baseURL + "/" + req.URL
			}
		}
	}

	// Convert req.Vars to map[string]any
	reqVars := make(map[string]any)
	if req.Vars != nil {
		for k, v := range req.Vars {
			reqVars[k] = v.Value
		}
	}

	// Convert env.Vars to map[string]any
	envVars := make(map[string]any)
	if env.Vars != nil {
		for k, v := range env.Vars {
			envVars[k] = v.Value
		}
	}

	varLayers := []map[string]any{
		reqVars,
		envVars,
	}
	resolvedURL, _ := model.ResolveVars(fullURL, varLayers...)
	if resolvedURL == "" {
		return errors.New("empty URL after resolution")
	}
	fullURL = resolvedURL

	var body io.Reader = nil
	if len(req.Body) > 0 {
		bodyStr, _ := model.ResolveVars(string(req.Body), varLayers...)
		body = bytes.NewReader([]byte(bodyStr))
	}

	httpReq, err := http.NewRequest(req.Method, fullURL, body)
	if err != nil {
		return &ExecutorError{Kind: "network", Cause: err}
	}

	for k, v := range mergedHeaders {
		resolvedV, _ := model.ResolveVars(v, varLayers...)
		httpReq.Header.Set(k, resolvedV)
	}

	if req.ContentType != "" && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", req.ContentType)
	}

	ApplyAuth(httpReq, env)

	if len(req.Params) > 0 {
		q := httpReq.URL.Query()
		for k, v := range req.Params {
			resolvedV, _ := model.ResolveVars(v, varLayers...)
			q.Add(k, resolvedV)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	req.ExecutedAt = time.Now()

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			return &ExecutorError{Kind: "timeout"}
		}
		return &ExecutorError{Kind: "network", Cause: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ExecutorError{Kind: "network", Cause: err}
	}

	responseHeaders := make(map[string]string)
	for k := range resp.Header {
		responseHeaders[k] = resp.Header.Get(k)
	}

	var parsedBody []byte
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var jsonBody any
		if err := json.Unmarshal(respBody, &jsonBody); err == nil {
			parsedBody, _ = json.MarshalIndent(jsonBody, "", "  ")
		}
	}

	req.Response = &model.Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    responseHeaders,
		Body:       parsedBody,
		RawBody:    respBody,
	}
	req.Duration = time.Since(req.ExecutedAt)

	return nil
}
