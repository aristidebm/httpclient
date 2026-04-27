package model

import (
	"strings"
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

type Environment struct {
	Name    string
	BaseURL string
	Headers map[string]string
	Vars    Variables
	Auth    *AuthConfig
}

func (e *Environment) Clone() *Environment {
	headers := make(map[string]string)
	for k, v := range e.Headers {
		headers[k] = v
	}
	vars := make(Variables)
	for k, v := range e.Vars {
		vars[k] = v
	}
	var auth *AuthConfig
	if e.Auth != nil {
		auth = &AuthConfig{
			Type:         e.Auth.Type,
			Username:     e.Auth.Username,
			Password:     e.Auth.Password,
			Token:        e.Auth.Token,
			TokenType:    e.Auth.TokenType,
			HeaderName:   e.Auth.HeaderName,
			ClientID:     e.Auth.ClientID,
			ClientSecret: e.Auth.ClientSecret,
			TokenURL:     e.Auth.TokenURL,
			AccessToken:  e.Auth.AccessToken,
			RefreshToken: e.Auth.RefreshToken,
			ExpiresAt:    e.Auth.ExpiresAt,
		}
	}
	return &Environment{
		Name:    e.Name,
		BaseURL: e.BaseURL,
		Headers: headers,
		Vars:    vars,
		Auth:    auth,
	}
}

func (e *Environment) Resolve(key string) (any, bool) {
	if v, ok := e.Vars.Get(key); ok {
		return v.Value, true
	}
	if v, ok := e.Headers[key]; ok {
		return v, true
	}
	return nil, false
}

func (e *Environment) SetBaseURL(url string) {
	if url != "" && !strings.HasSuffix(url, "/") {
		url = url + "/"
	}
	e.BaseURL = url
}
