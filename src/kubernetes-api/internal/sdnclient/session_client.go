package sdnclient

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"time"
)

// SessionClient wraps around http.Client and automatically adds authorization headers.
type SessionClient struct {
	httpClient *http.Client
	BaseURL    string
	AuthToken  string
}

// NewSessionClient creates a new SessionClient with basic auth credentials.
func NewSessionClient(baseURL, username, password string) *SessionClient {
	authToken := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return &SessionClient{
		httpClient: &http.Client{Timeout: time.Second * 10},
		BaseURL:    baseURL,
		AuthToken:  authToken,
	}
}

// newRequest creates a new HTTP request with the necessary authentication headers.
func (c *SessionClient) newRequest(method, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, c.BaseURL+url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Basic "+c.AuthToken)
	req.Header.Add("Content-Type", "application/json")
	return req, nil
}

// Do sends an HTTP request and returns an HTTP response, similar to http.Client's Do.
func (c *SessionClient) Do(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}

// Get wraps the GET method with authorization.
func (c *SessionClient) Get(url string) (*http.Response, error) {
	req, err := c.newRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post wraps the POST method with authorization.
func (c *SessionClient) Post(url string, body []byte) (*http.Response, error) {
	req, err := c.newRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Delete wraps the DELETE method with authorization.
func (c *SessionClient) Delete(url string) (*http.Response, error) {
	req, err := c.newRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}
