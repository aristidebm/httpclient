package model

import (
	"fmt"
	"time"
)

type Session struct {
	ID              string
	Name            string
	EnvName         string
	ParentID        string
	Requests        []*Request
	HeaderOverrides map[string]string
	VarOverrides    map[string]any
	OpenAPISpec     *OpenAPISpec // stored as pointer for JSON serialization
	CreatedAt       time.Time
}

type OpenAPISpec struct {
	Title   string
	Version string
	Routes  []Route
}

type Route struct {
	Method  string
	Path    string
	Summary string
	Tags    []string
	Params  []Parameter
}

type Parameter struct {
	Name     string
	In       string
	Required bool
}

func (s *Session) NextRequestID() string {
	return fmt.Sprintf("r%d", len(s.Requests)+1)
}

func (s *Session) GetRequest(id string) (*Request, bool) {
	for _, r := range s.Requests {
		if r.ID == id {
			return r, true
		}
	}
	return nil, false
}

func (s *Session) AddRequest(r *Request) {
	r.ID = s.NextRequestID()
	s.Requests = append(s.Requests, r)
}

func (s *Session) RemoveRequest(id string) bool {
	for i, r := range s.Requests {
		if r.ID == id {
			s.Requests = append(s.Requests[:i], s.Requests[i+1:]...)
			return true
		}
	}
	return false
}
