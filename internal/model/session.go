package model

import (
	"fmt"
	"time"
)

type AuthConfig struct {
	Type string // "basic", "token", "oauth"

	// Basic auth
	Username string
	Password string

	// Token auth
	Token      string
	TokenType  string // "Bearer", "Token", or custom
	HeaderName string // default: "Authorization"

	// OAuth
	ClientID     string
	ClientSecret string
	TokenURL     string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type Session struct {
	ID          string
	Name        string
	ParentID    string
	BaseURL     string // session-specific base URL
	Requests    []*Request
	Headers     map[string]string // per-session headers
	Vars        Variables
	OpenAPISpec *OpenAPISpec // stored as pointer for JSON serialization
	Auth        *AuthConfig  // session-level auth
	CreatedAt   time.Time
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
	// Clone headers
	headers := make(map[string]string)
	for k, v := range s.Headers {
		headers[k] = v
	}

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
		ID:          s.ID,
		Name:        s.Name,
		ParentID:    s.ParentID,
		BaseURL:     s.BaseURL,
		Requests:    requests,
		Headers:     headers,
		Vars:        vars,
		OpenAPISpec: s.OpenAPISpec,
		Auth:        auth,
		CreatedAt:   s.CreatedAt,
	}
}
