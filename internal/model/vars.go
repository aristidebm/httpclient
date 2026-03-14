package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var varPattern = regexp.MustCompile(`\{([^}]+)\}`)

type VarScope string

const (
	VarScopeEnv     VarScope = "env"
	VarScopeSession VarScope = "session"
	VarScopeShell   VarScope = "shell"
)

type Variable struct {
	Name    string
	Value   any
	Scope   VarScope
	Public  bool // if false, variable doesn't appear in listings
	Created time.Time
	Updated time.Time
}

type Variables map[string]*Variable

func (v Variables) Set(key string, value any, scope VarScope) {
	now := time.Now()
	if existing, ok := v[key]; ok {
		existing.Value = value
		existing.Scope = scope
		existing.Updated = now
	} else {
		v[key] = &Variable{
			Name:    key,
			Value:   value,
			Scope:   scope,
			Public:  true,
			Created: now,
			Updated: now,
		}
	}
}

func (v Variables) SetPublic(key string, public bool) {
	if v == nil {
		return
	}
	if existing, ok := v[key]; ok {
		existing.Public = public
		existing.Updated = time.Now()
	}
}

func (v Variables) Get(key string) (*Variable, bool) {
	variable, ok := v[key]
	return variable, ok
}

func (v Variables) Delete(key string) {
	delete(v, key)
}

func (v Variables) List() []*Variable {
	vars := make([]*Variable, 0, len(v))
	for _, v := range v {
		vars = append(vars, v)
	}
	return vars
}

func (v Variables) ListByScope(scope VarScope) []*Variable {
	var vars []*Variable
	for _, variable := range v {
		if variable.Scope == scope {
			vars = append(vars, variable)
		}
	}
	return vars
}

func (v Variables) ListPublic() []*Variable {
	var vars []*Variable
	for _, variable := range v {
		if variable.Public {
			vars = append(vars, variable)
		}
	}
	return vars
}

func ResolveVars(template string, layers ...map[string]any) (string, []string) {
	var unresolved []string

	result := varPattern.ReplaceAllStringFunc(template, func(match string) string {
		key := match[1 : len(match)-1]

		for i := 0; i < len(layers); i++ {
			layer := layers[i]
			if layer == nil {
				continue
			}
			if val, ok := layer[key]; ok {
				return fmt.Sprintf("%v", val)
			}
		}

		unresolved = append(unresolved, key)
		return match
	})

	return result, unresolved
}

func ResolveVarsStrict(template string, layers ...map[string]any) (string, error) {
	result, unresolved := ResolveVars(template, layers...)
	if len(unresolved) > 0 {
		return result, errors.New("unresolved variables: " + strings.Join(unresolved, ", "))
	}
	return result, nil
}
