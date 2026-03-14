package model

import "time"

type SessionTree struct {
	Sessions     map[string]*Session
	CurrentID    string
	PreviousID   string
	Environments map[string]*Environment
}

func NewSessionTree() *SessionTree {
	env := &Environment{
		Name:    "local",
		BaseURL: "",
		Headers: make(map[string]string),
		Vars:    make(map[string]any),
	}
	sess := &Session{
		ID:              "default",
		Name:            "default",
		EnvName:         "local",
		ParentID:        "",
		Requests:        make([]*Request, 0),
		HeaderOverrides: make(map[string]string),
		VarOverrides:    make(map[string]any),
		CreatedAt:       time.Now(),
	}
	return &SessionTree{
		Sessions: map[string]*Session{
			"default": sess,
		},
		CurrentID: "default",
		Environments: map[string]*Environment{
			"local": env,
		},
	}
}

func (t *SessionTree) Current() *Session {
	return t.Sessions[t.CurrentID]
}

func (t *SessionTree) CurrentEnv() *Environment {
	sess := t.Current()
	if sess == nil {
		return nil
	}
	return t.Environments[sess.EnvName]
}

func (t *SessionTree) Children(sessionID string) []*Session {
	var result []*Session
	for _, s := range t.Sessions {
		if s.ParentID == sessionID {
			result = append(result, s)
		}
	}
	return result
}
