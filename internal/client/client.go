package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const maxResponseBodyBytes int64 = 16 * 1024 * 1024
const maxTLSCAFileBytes int64 = 1024 * 1024

// Client is an HTTP client for the LabTether v2 API.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// New creates a new API client.
func New(baseURL, apiKey string) *Client {
	client, _ := NewWithTLSCAFile(baseURL, apiKey, "")
	return client
}

// NewWithTLSCAFile creates an API client that additionally trusts the PEM CA
// certificates in caFile. This is the secure connection path for LabTether's
// default private CA and avoids a global trust-store change or TLS bypass.
func NewWithTLSCAFile(baseURL, apiKey, caFile string) (*Client, error) {
	trimmedBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	httpClient := &http.Client{
		Timeout: 5 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			origin, err := url.Parse(trimmedBaseURL)
			if err != nil || !sameOrigin(origin, req.URL) {
				return fmt.Errorf("refusing cross-origin redirect to %s", req.URL.Redacted())
			}
			return nil
		},
	}

	if strings.TrimSpace(caFile) != "" {
		rootCAs, err := loadTLSRootCAs(caFile)
		if err != nil {
			return nil, err
		}
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    rootCAs,
		}
		httpClient.Transport = transport
	}

	return &Client{
		BaseURL:    trimmedBaseURL,
		APIKey:     strings.TrimSpace(apiKey),
		HTTPClient: httpClient,
	}, nil
}

// ValidateTLSCAFile verifies that path is a bounded regular PEM file with at
// least one CA certificate. It does not alter the machine-wide trust store.
func ValidateTLSCAFile(path string) error {
	_, err := loadTLSRootCAs(path)
	return err
}

func loadTLSRootCAs(path string) (*x509.CertPool, error) {
	file, err := os.Open(strings.TrimSpace(path)) // #nosec G304 -- The operator explicitly chooses the local CA bundle to trust.
	if err != nil {
		return nil, fmt.Errorf("read TLS CA file: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect TLS CA file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("TLS CA file must be a regular file")
	}
	pemData, err := io.ReadAll(io.LimitReader(file, maxTLSCAFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read TLS CA file: %w", err)
	}
	if int64(len(pemData)) > maxTLSCAFileBytes {
		return nil, fmt.Errorf("TLS CA file exceeds %d byte limit", maxTLSCAFileBytes)
	}
	rootCAs, err := x509.SystemCertPool()
	if err != nil || rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	if !rootCAs.AppendCertsFromPEM(pemData) {
		return nil, fmt.Errorf("TLS CA file contains no valid PEM certificates")
	}
	return rootCAs, nil
}

// ValidateBaseURL rejects credential-bearing requests over plaintext except
// for an explicit loopback address used by local development.
func ValidateBaseURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("invalid hub URL")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("hub URL must not include a query or fragment")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		return nil
	case "http":
		if isLoopbackHost(parsed.Hostname()) {
			return nil
		}
		return fmt.Errorf("refusing to send API credentials over plaintext HTTP to non-loopback host %q", parsed.Hostname())
	default:
		return fmt.Errorf("hub URL must use HTTPS (HTTP is allowed only for loopback)")
	}
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func sameOrigin(a, b *url.URL) bool {
	if a == nil || b == nil {
		return false
	}
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
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
	if err := ValidateBaseURL(c.BaseURL); err != nil {
		return nil, err
	}
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

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if int64(len(respBody)) > maxResponseBodyBytes {
		return nil, fmt.Errorf("response exceeds %d byte limit", maxResponseBodyBytes)
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
