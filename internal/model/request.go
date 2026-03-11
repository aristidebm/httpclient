package model

import (
	"encoding/json"
	"time"
)

type Duration int64

func (d Duration) Duration() time.Duration {
	return time.Duration(d) * time.Millisecond
}

func ParseTime(s string) (time.Time, time.Time) {
	if s == "" {
		return time.Time{}, time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, time.Time{}
	}
	return t, t
}

func NewDuration(d time.Duration) Duration {
	return Duration(d.Milliseconds())
}

type Request struct {
	ID          string
	Method      string
	URL         string
	Headers     map[string]string
	Params      map[string]string
	Body        []byte
	ContentType string
	Response    *Response
	ExecutedAt  time.Time
	Duration    time.Duration
	Note        string
	Vars        map[string]any
}

type Response struct {
	StatusCode int
	Status     string
	Headers    map[string]string
	Body       []byte
	RawBody    []byte
}

func (r *Response) MarshalJSON() ([]byte, error) {
	type Alias Response
	aux := &struct {
		RawBody string `json:"rawBody"`
		*Alias
	}{
		Alias:   (*Alias)(r),
		RawBody: "",
	}
	if len(r.RawBody) > 0 {
		aux.RawBody = string(r.RawBody)
	}
	return json.Marshal(aux)
}

func (r *Response) UnmarshalJSON(data []byte) error {
	type Alias Response
	aux := &struct {
		RawBody string `json:"rawBody"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.RawBody != "" {
		r.RawBody = []byte(aux.RawBody)
	}
	return nil
}

func (r *Request) Clone() *Request {
	headers := make(map[string]string)
	for k, v := range r.Headers {
		headers[k] = v
	}
	params := make(map[string]string)
	for k, v := range r.Params {
		params[k] = v
	}
	body := make([]byte, len(r.Body))
	copy(body, r.Body)
	vars := make(map[string]any)
	for k, v := range r.Vars {
		vars[k] = v
	}
	return &Request{
		ID:          r.ID,
		Method:      r.Method,
		URL:         r.URL,
		Headers:     headers,
		Params:      params,
		Body:        body,
		ContentType: r.ContentType,
		Response:    nil,
		ExecutedAt:  time.Time{},
		Duration:    0,
		Note:        r.Note,
		Vars:        vars,
	}
}

func (r *Request) IsExecuted() bool {
	return r.Response != nil
}
