package v2

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Scope3APIClient struct {
	httpClient *http.Client
	baseUrl    string
	apiKey     string
}

type Scope3APIClientConfig struct {
	Host               string
	ApiKey             string
	Timeout            time.Duration
	MaxIdleConnections int
	IdleConnTimeout    time.Duration
}

func NewScope3APIClient(config Scope3APIClientConfig) *Scope3APIClient {
	baseUrl := config.Host
	if !strings.HasPrefix(baseUrl, "http") {
		baseUrl = "https://" + baseUrl
	}
	baseUrl += "/v2"

	client := &http.Client{
		Timeout: config.Timeout, // Set a timeout for the request
		Transport: &http.Transport{
			MaxIdleConns:    config.MaxIdleConnections,
			IdleConnTimeout: config.IdleConnTimeout,
		},
	}
	return &Scope3APIClient{
		httpClient: client,
		baseUrl:    baseUrl,
		apiKey:     config.ApiKey,
	}
}

type Scope3ServerError struct {
	Message string
	Err     error
}

func (e Scope3ServerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("Scope3 server error: %s, caused by: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("Scope3 server error: %s", e.Message)
}

func (s *Scope3APIClient) doPost(url string, requestBodyBytes []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+s.apiKey)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	return s.httpClient.Do(req)
}
