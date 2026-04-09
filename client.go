package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const BaseURL = "https://api.octa.space"

// Client is an HTTP client for the OctaSpace API.
type Client struct {
	apiKey string
	http   *http.Client
}

// NewClient creates a new API client authenticated with the given API key.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) get(path string, dst any) error {
	req, err := http.NewRequest(http.MethodGet, BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("authorization", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, body)
	}
	return json.Unmarshal(body, dst)
}

func (c *Client) getRaw(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("authorization", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, body)
	}
	return body, nil
}

func (c *Client) post(path string, reqBody, dst any) error {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, body)
	}
	return json.Unmarshal(body, dst)
}
