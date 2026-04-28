package model

import "time"

type SessionTree struct {
	Sessions   map[string]*Session
	CurrentID  string
	PreviousID string
}

func NewSessionTree() *SessionTree {
	sess := &Session{
		ID:        "default",
		Name:      "default",
		ParentID:  "",
		BaseURL:   "",
		Requests:  make([]*Request, 0),
		Headers:   make(map[string]string),
		Vars:      make(Variables),
		CreatedAt: time.Now(),
	}
	return &SessionTree{
		Sessions: map[string]*Session{
			"default": sess,
		},
		CurrentID: "default",
	}
}

func (t *SessionTree) Current() *Session {
	return t.Sessions[t.CurrentID]
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

// GetInheritedHeaders returns headers merged from all ancestors (parent -> grandparent -> ...)
func (t *SessionTree) GetInheritedHeaders(sessionID string) map[string]string {
	result := make(map[string]string)
	sess := t.Sessions[sessionID]
	if sess == nil {
		return result
	}

	// Collect headers from all ancestors
	visited := make(map[string]bool)
	current := sess
	for current != nil && current.ParentID != "" && !visited[current.ParentID] {
		visited[current.ParentID] = true
		parent := t.Sessions[current.ParentID]
		if parent == nil {
			break
		}
		// Merge parent headers (ancestors override earlier ones)
		for k, v := range parent.Headers {
			result[k] = v
		}
		current = parent
	}

	return result
}

// GetInheritedVars returns variables merged from all ancestors
func (t *SessionTree) GetInheritedVars(sessionID string) Variables {
	result := make(Variables)
	sess := t.Sessions[sessionID]
	if sess == nil {
		return result
	}

	visited := make(map[string]bool)
	current := sess
	for current != nil && current.ParentID != "" && !visited[current.ParentID] {
		visited[current.ParentID] = true
		parent := t.Sessions[current.ParentID]
		if parent == nil {
			break
		}
		for k, v := range parent.Vars {
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}
		current = parent
	}

	return result
}

// GetInheritedAuth returns the first auth found in the ancestor chain
func (t *SessionTree) GetInheritedAuth(sessionID string) *AuthConfig {
	sess := t.Sessions[sessionID]
	if sess == nil {
		return nil
	}

	visited := make(map[string]bool)
	current := sess
	for current != nil && !visited[current.ID] {
		if current.Auth != nil {
			return current.Auth
		}
		visited[current.ID] = true
		if current.ParentID == "" {
			break
		}
		current = t.Sessions[current.ParentID]
	}

	return nil
}

// GetInheritedBaseURL returns the first BaseURL found in the ancestor chain
func (t *SessionTree) GetInheritedBaseURL(sessionID string) string {
	sess := t.Sessions[sessionID]
	if sess == nil {
		return ""
	}

	visited := make(map[string]bool)
	current := sess
	for current != nil && !visited[current.ID] {
		if current.BaseURL != "" {
			return current.BaseURL
		}
		visited[current.ID] = true
		if current.ParentID == "" {
			break
		}
		current = t.Sessions[current.ParentID]
	}

	return ""
}

// GetEffectiveHeaders returns headers with inheritance (session + ancestors)
func (t *SessionTree) GetEffectiveHeaders(sessionID string) map[string]string {
	inherited := t.GetInheritedHeaders(sessionID)
	sess := t.Sessions[sessionID]
	if sess == nil {
		return inherited
	}

	// Session headers override inherited
	for k, v := range sess.Headers {
		inherited[k] = v
	}

	return inherited
}

// GetEffectiveVars returns variables with inheritance (session + ancestors)
func (t *SessionTree) GetEffectiveVars(sessionID string) Variables {
	inherited := t.GetInheritedVars(sessionID)
	sess := t.Sessions[sessionID]
	if sess == nil {
		return inherited
	}

	// Session vars override inherited
	for k, v := range sess.Vars {
		inherited[k] = v
	}

	return inherited
}
