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
	Vars            Variables
	OpenAPISpec     *OpenAPISpec // stored as pointer for JSON serialization
	Auth            *AuthConfig  // session-level auth (overrides env auth)
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

func (s *Session) Clone() *Session {
	// Clone vars
	vars := make(Variables)
	for k, v := range s.Vars {
		vars[k] = v
	}

	// Clone requests
	requests := make([]*Request, len(s.Requests))
	for i, r := range s.Requests {
		requests[i] = r.Clone()
	}

	// Clone auth
	var auth *AuthConfig
	if s.Auth != nil {
		auth = &AuthConfig{
			Type:         s.Auth.Type,
			Username:     s.Auth.Username,
			Password:     s.Auth.Password,
			Token:        s.Auth.Token,
			TokenType:    s.Auth.TokenType,
			HeaderName:   s.Auth.HeaderName,
			ClientID:     s.Auth.ClientID,
			ClientSecret: s.Auth.ClientSecret,
			TokenURL:     s.Auth.TokenURL,
			AccessToken:  s.Auth.AccessToken,
			RefreshToken: s.Auth.RefreshToken,
			ExpiresAt:    s.Auth.ExpiresAt,
		}
	}

	return &Session{
		ID:              s.ID,
		Name:            s.Name,
		EnvName:         s.EnvName,
		ParentID:        s.ParentID,
		Requests:        requests,
		HeaderOverrides: s.HeaderOverrides,
		Vars:            vars,
		OpenAPISpec:     s.OpenAPISpec,
		Auth:            auth,
		CreatedAt:       s.CreatedAt,
	}
}
