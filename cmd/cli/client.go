package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const httpTimeout = 10 * time.Second

// ClientOptions configures the HTTP client.
type ClientOptions struct {
	BaseURL   string
	Token     string
	ProjectID string
}

// Client calls the CasperCloud API.
type Client struct {
	base      string
	http      *http.Client
	Token     string
	ProjectID uuid.UUID
}

// NewClient builds an API client. ProjectID may be zero for unscoped calls (e.g. login).
func NewClient(opt ClientOptions) (*Client, error) {
	var pid uuid.UUID
	if strings.TrimSpace(opt.ProjectID) != "" {
		p, err := uuid.Parse(opt.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("invalid project_id in config: %w", err)
		}
		pid = p
	}
	return &Client{
		base:      strings.TrimRight(strings.TrimSpace(opt.BaseURL), "/"),
		Token:     opt.Token,
		ProjectID: pid,
		http: &http.Client{
			Timeout: httpTimeout,
		},
	}, nil
}

func (c *Client) absURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return c.base + path
}

// APIStatusError is returned for non-2xx API responses.
type APIStatusError struct {
	Status int
	Body   []byte
}

func (e APIStatusError) Error() string {
	var env struct {
		Error struct {
			Message string         `json:"message"`
			Fields  map[string]any `json:"fields"`
		} `json:"error"`
	}
	if json.Unmarshal(e.Body, &env) == nil && env.Error.Message != "" {
		return fmt.Sprintf("%s (%d)", env.Error.Message, e.Status)
	}
	s := strings.TrimSpace(string(e.Body))
	if s == "" {
		return fmt.Sprintf("HTTP %d", e.Status)
	}
	return fmt.Sprintf("HTTP %d: %s", e.Status, s)
}

func (c *Client) do(method, path string, body any, withAuth bool, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.absURL(path), rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if withAuth && c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return APIStatusError{Status: resp.StatusCode, Body: raw}
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func (c *Client) projectPath(suffix string) string {
	if c.ProjectID == uuid.Nil {
		return ""
	}
	s := suffix
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	return fmt.Sprintf("/v1/projects/%s%s", c.ProjectID.String(), s)
}
