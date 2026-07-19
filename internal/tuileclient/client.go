package tuileclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client talks to a running Tuile HTTP API.
type Client struct {
	BaseURL string
	HTTP    *http.Client
	Boot    string
}

// New returns a client with defaults.
func New(baseURL, bootstrap string) *Client {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:7710"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 0},
		Boot:    bootstrap,
	}
}

type apiError struct {
	Error string `json:"error"`
}

type createResponse struct {
	SessionID string `json:"session_id"`
	Workspace string `json:"workspace"`
	Token     string `json:"token"`
}

type sessionInfo struct {
	SessionID string `json:"session_id"`
	Workspace string `json:"workspace"`
	Cols      int    `json:"cols"`
	Rows      int    `json:"rows"`
}

// SessionSummary is a listed session row.
type SessionSummary = sessionInfo

// CreateSession starts a new PTY session.
func (c *Client) CreateSession(workspace, cli string) (createResponse, error) {
	body := map[string]string{"workspace": workspace}
	if cli != "" {
		body["cli"] = cli
	}
	var out createResponse
	if err := c.postJSON("/v1/sessions", c.bootHeaders(), body, http.StatusCreated, &out); err != nil {
		return createResponse{}, err
	}
	return out, nil
}

// ListSessions returns active sessions.
func (c *Client) ListSessions() ([]SessionSummary, error) {
	var out struct {
		Sessions []SessionSummary `json:"sessions"`
	}
	if err := c.getJSON("/v1/sessions", c.bootHeaders(), &out); err != nil {
		return nil, err
	}
	return out.Sessions, nil
}

// CloseSession stops a session.
func (c *Client) CloseSession(sessionID string) error {
	req, err := http.NewRequest(http.MethodDelete, c.url("/v1/sessions/"+url.PathEscape(sessionID)), nil)
	if err != nil {
		return err
	}
	for k, v := range c.bootHeaders() {
		req.Header.Set(k, v)
	}
	res, err := c.do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		return c.decodeAPIError(res)
	}
	return nil
}

// SendInput writes PTY input. A trailing newline is sent as Enter unless raw is true.
func (c *Client) SendInput(sessionID, token, input string, raw bool) error {
	body := map[string]any{"input": input, "raw": raw}
	return c.postJSON("/v1/sessions/"+url.PathEscape(sessionID)+"/input", c.sessionHeaders(token), body, http.StatusOK, nil)
}

// ScreenOptions selects compact screen reads.
type ScreenOptions struct {
	Format string
	Tail   int
	Since  uint64
}

type screenTextResponse struct {
	Version uint64 `json:"version"`
	Text    string `json:"text"`
}

// ReadScreenText returns compact tail text for agents.
func (c *Client) ReadScreenText(sessionID, token string, opts ScreenOptions) (screenTextResponse, error) {
	q := url.Values{}
	if opts.Format != "" {
		q.Set("format", opts.Format)
	} else {
		q.Set("format", "text")
	}
	if opts.Tail > 0 {
		q.Set("tail", strconv.Itoa(opts.Tail))
	}
	if opts.Since > 0 {
		q.Set("since", strconv.FormatUint(opts.Since, 10))
	}
	path := "/v1/sessions/" + url.PathEscape(sessionID) + "/screen"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	var out screenTextResponse
	if err := c.getJSON(path, c.sessionHeaders(token), &out); err != nil {
		return screenTextResponse{}, err
	}
	return out, nil
}

// WaitRequest is the body for POST /wait.
type WaitRequest struct {
	Contains  string `json:"contains,omitempty"`
	Since     uint64 `json:"since,omitempty"`
	TimeoutMS int    `json:"timeout_ms,omitempty"`
	Tail      int    `json:"tail,omitempty"`
}

// WaitResponse is returned by Wait.
type WaitResponse struct {
	Version uint64 `json:"version"`
	Matched bool   `json:"matched"`
	Text    string `json:"text"`
}

// Wait blocks until output matches or timeout.
func (c *Client) Wait(sessionID, token string, req WaitRequest) (WaitResponse, error) {
	var out WaitResponse
	if err := c.postJSON("/v1/sessions/"+url.PathEscape(sessionID)+"/wait", c.sessionHeaders(token), req, http.StatusOK, &out); err != nil {
		return WaitResponse{}, err
	}
	return out, nil
}

// WaitWithTimeout uses a bounded HTTP client timeout for long polls.
func (c *Client) WaitWithTimeout(sessionID, token string, req WaitRequest, timeout time.Duration) (WaitResponse, error) {
	prev := c.HTTP
	if timeout > 0 {
		c.HTTP = &http.Client{Timeout: timeout + 5*time.Second}
		defer func() { c.HTTP = prev }()
	}
	return c.Wait(sessionID, token, req)
}

func (c *Client) url(path string) string {
	return c.BaseURL + path
}

func (c *Client) bootHeaders() map[string]string {
	return map[string]string{"Authorization": "Bearer " + c.Boot}
}

func (c *Client) sessionHeaders(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(req)
}

func (c *Client) getJSON(path string, headers map[string]string, out any) error {
	req, err := http.NewRequest(http.MethodGet, c.url(path), nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := c.do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotModified {
		return nil
	}
	if res.StatusCode != http.StatusOK {
		return c.decodeAPIError(res)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (c *Client) postJSON(path string, headers map[string]string, body any, wantStatus int, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.url(path), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := c.do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != wantStatus {
		return c.decodeAPIError(res)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (c *Client) decodeAPIError(res *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	var ae apiError
	if json.Unmarshal(data, &ae) == nil && ae.Error != "" {
		return fmt.Errorf("tuile api %s: %s", res.Status, ae.Error)
	}
	if len(data) > 0 {
		return fmt.Errorf("tuile api %s: %s", res.Status, strings.TrimSpace(string(data)))
	}
	return fmt.Errorf("tuile api %s", res.Status)
}
