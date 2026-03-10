package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func New(controlPort string) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%s", controlPort),
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

type MountRequest struct {
	Path  string `json:"path"`
	Route string `json:"route"`
}

type UnmountRequest struct {
	Route string `json:"route"`
}

type MountInfo struct {
	Route     string `json:"route"`
	LocalPath string `json:"local_path"`
	Type      string `json:"type"`
	Pattern   string `json:"pattern"`
}

type APIResponse struct {
	Error string          `json:"error,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
	OK    bool            `json:"ok"`
}

func (c *Client) Mount(path, route string) (*MountInfo, error) {
	body, err := json.Marshal(MountRequest{Path: path, Route: route})
	if err != nil {
		return nil, err
	}

	resp, err := c.do("POST", "/v1/mounts", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("server is not running (start it with `serve start`): %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("%s", apiResp.Error)
	}

	var info MountInfo
	if err := json.Unmarshal(apiResp.Data, &info); err != nil {
		return nil, fmt.Errorf("invalid response data: %w", err)
	}

	return &info, nil
}

func (c *Client) Unmount(route string) error {
	body, err := json.Marshal(UnmountRequest{Route: route})
	if err != nil {
		return err
	}

	resp, err := c.do("DELETE", "/v1/mounts", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("server is not running (start it with `serve start`): %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}

	if !apiResp.OK {
		return fmt.Errorf("%s", apiResp.Error)
	}

	return nil
}

func (c *Client) List() ([]MountInfo, error) {
	resp, err := c.do("GET", "/v1/mounts", nil)
	if err != nil {
		return nil, fmt.Errorf("server is not running (start it with `serve start`): %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("%s", apiResp.Error)
	}

	var mounts []MountInfo
	if err := json.Unmarshal(apiResp.Data, &mounts); err != nil {
		return nil, fmt.Errorf("invalid response data: %w", err)
	}

	return mounts, nil
}

func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.httpClient.Do(req)
}
