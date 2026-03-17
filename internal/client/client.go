package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the LabTether v2 API.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// New creates a new API client.
func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// V2Response is the standard v2 response envelope.
type V2Response struct {
	RequestID string          `json:"request_id"`
	Data      json.RawMessage `json:"data"`
	Error     string          `json:"error,omitempty"`
	Message   string          `json:"message,omitempty"`
	Status    int             `json:"status,omitempty"`
	Meta      *V2Meta         `json:"meta,omitempty"`
}

type V2Meta struct {
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

func (c *Client) do(method, path string, body any) (*V2Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var v2resp V2Response
	if err := json.Unmarshal(respBody, &v2resp); err != nil {
		// Non-JSON response
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	if resp.StatusCode >= 400 {
		msg := v2resp.Message
		if msg == "" {
			msg = v2resp.Error
		}
		return &v2resp, fmt.Errorf("%s (status %d)", msg, resp.StatusCode)
	}

	return &v2resp, nil
}

// Get performs a GET request.
func (c *Client) Get(path string) (*V2Response, error) {
	return c.do("GET", path, nil)
}

// Post performs a POST request.
func (c *Client) Post(path string, body any) (*V2Response, error) {
	return c.do("POST", path, body)
}

// Put performs a PUT request.
func (c *Client) Put(path string, body any) (*V2Response, error) {
	return c.do("PUT", path, body)
}

// Patch performs a PATCH request.
func (c *Client) Patch(path string, body any) (*V2Response, error) {
	return c.do("PATCH", path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) (*V2Response, error) {
	return c.do("DELETE", path, nil)
}
