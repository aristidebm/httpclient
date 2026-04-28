package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"httpclient/internal/model"
)

var (
	ConfigDir    = "~/.httpclient"
	SessionsFile = "~/.httpclient/sessions.json"
	ConfigFile   = "~/.httpclient/config.toml"
)

func expandPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty path")
	}
	if path[:1] == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot find home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}
	return path, nil
}

func ensureConfigDir() error {
	path, err := expandPath(ConfigDir)
	if err != nil {
		return err
	}
	return os.MkdirAll(path, 0755)
}

type ResponseForJSON struct {
	StatusCode int               `json:"statusCode"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
	RawBody    string            `json:"rawBody"`
}

type RequestForJSON struct {
	ID          string            `json:"id"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
	Params      map[string]string `json:"params"`
	Body        []byte            `json:"body"`
	ContentType string            `json:"contentType"`
	Response    *ResponseForJSON  `json:"response,omitempty"`
	ExecutedAt  string            `json:"executedAt"`
	Duration    int64             `json:"duration"`
	Note        string            `json:"note"`
}

type SessionForJSON struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	ParentID  string            `json:"parentId"`
	BaseURL   string            `json:"baseUrl,omitempty"`
	Requests  []*RequestForJSON `json:"requests"`
	Headers   map[string]string `json:"headers,omitempty"`
	Vars      model.Variables   `json:"vars"`
	CreatedAt string            `json:"createdAt"`
}

type SessionTreeForJSON struct {
	Sessions  map[string]*SessionForJSON `json:"sessions"`
	CurrentID string                     `json:"currentId"`
}

func SaveTree(tree *model.SessionTree) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}

	path, err := expandPath(SessionsFile)
	if err != nil {
		return err
	}

	jsonTree := SessionTreeForJSON{
		Sessions:  make(map[string]*SessionForJSON),
		CurrentID: tree.CurrentID,
	}

	for id, sess := range tree.Sessions {
		reqs := make([]*RequestForJSON, len(sess.Requests))
		for i, req := range sess.Requests {
			reqs[i] = &RequestForJSON{
				ID:          req.ID,
				Method:      req.Method,
				URL:         req.URL,
				Headers:     req.Headers,
				Params:      req.Params,
				Body:        req.Body,
				ContentType: req.ContentType,
				Response:    nil,
				ExecutedAt:  req.ExecutedAt.Format("2006-01-02T15:04:05Z07:00"),
				Duration:    int64(req.Duration),
				Note:        req.Note,
			}
			if req.Response != nil {
				reqs[i].Response = &ResponseForJSON{
					StatusCode: req.Response.StatusCode,
					Status:     req.Response.Status,
					Headers:    req.Response.Headers,
					Body:       req.Response.Body,
					RawBody:    string(req.Response.RawBody),
				}
			}
		}
		jsonTree.Sessions[id] = &SessionForJSON{
			ID:        sess.ID,
			Name:      sess.Name,
			ParentID:  sess.ParentID,
			BaseURL:   sess.BaseURL,
			Requests:  reqs,
			Headers:   sess.Headers,
			Vars:      sess.Vars,
			CreatedAt: sess.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	data, err := json.MarshalIndent(jsonTree, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tree: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func LoadTree() (*model.SessionTree, error) {
	path, err := expandPath(SessionsFile)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.NewSessionTree(), nil
		}
		return nil, err
	}

	var jsonTree SessionTreeForJSON
	if err := json.Unmarshal(data, &jsonTree); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: sessions.json is corrupt, starting fresh: %v\n", err)
		return model.NewSessionTree(), nil
	}

	tree := &model.SessionTree{
		Sessions:  make(map[string]*model.Session),
		CurrentID: jsonTree.CurrentID,
	}

	for id, jsonSess := range jsonTree.Sessions {
		createdAt, _ := model.ParseTime(jsonSess.CreatedAt)
		reqs := make([]*model.Request, len(jsonSess.Requests))
		for i, jsonReq := range jsonSess.Requests {
			executedAt, _ := model.ParseTime(jsonReq.ExecutedAt)
			var resp *model.Response
			if jsonReq.Response != nil {
				resp = &model.Response{
					StatusCode: jsonReq.Response.StatusCode,
					Status:     jsonReq.Response.Status,
					Headers:    jsonReq.Response.Headers,
					Body:       jsonReq.Response.Body,
					RawBody:    []byte(jsonReq.Response.RawBody),
				}
			}
			reqs[i] = &model.Request{
				ID:          jsonReq.ID,
				Method:      jsonReq.Method,
				URL:         jsonReq.URL,
				Headers:     jsonReq.Headers,
				Params:      jsonReq.Params,
				Body:        jsonReq.Body,
				ContentType: jsonReq.ContentType,
				Response:    resp,
				ExecutedAt:  executedAt,
				Duration:    time.Duration(jsonReq.Duration) * time.Millisecond,
				Note:        jsonReq.Note,
			}
		}
		tree.Sessions[id] = &model.Session{
			ID:        jsonSess.ID,
			Name:      jsonSess.Name,
			ParentID:  jsonSess.ParentID,
			BaseURL:   jsonSess.BaseURL,
			Requests:  reqs,
			Headers:   jsonSess.Headers,
			Vars:      jsonSess.Vars,
			CreatedAt: createdAt,
		}
	}

	if tree.Current() == nil {
		return model.NewSessionTree(), nil
	}

	return tree, nil
}
