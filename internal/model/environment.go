package model

import (
	"strings"
)

type Environment struct {
	Name    string
	BaseURL string
	Headers map[string]string
	Vars    Variables
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
	return &Environment{
		Name:    e.Name,
		BaseURL: e.BaseURL,
		Headers: headers,
		Vars:    vars,
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
