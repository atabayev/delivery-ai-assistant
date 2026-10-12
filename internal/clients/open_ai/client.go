package openai

import (
	"net/http"
	"time"
)

type Client struct {
	httpClient    *http.Client
	baseURL       string
	apiKey        string
	model         string
	timeout       time.Duration
	streamTimeout time.Duration
}

func New(
	httpClient *http.Client,
	baseURL string,
	model string,
	apiKey string,
	timeout time.Duration,
	streamTimeout time.Duration,
) *Client {
	return &Client{
		httpClient:    httpClient,
		baseURL:       baseURL,
		model:         model,
		apiKey:        apiKey,
		timeout:       timeout,
		streamTimeout: streamTimeout,
	}
}
