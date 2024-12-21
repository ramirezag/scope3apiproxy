package v2

import (
	"bytes"
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
	if !strings.HasPrefix(baseUrl, "https://") {
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
