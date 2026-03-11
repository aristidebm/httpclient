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
	CreatedAt       time.Time
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
