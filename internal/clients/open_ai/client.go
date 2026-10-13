package openai

import (
	"delivery-ai-assistant/internal/config"
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
	cfg config.LLM,
) *Client {
	return &Client{
		httpClient:    httpClient,
		baseURL:       cfg.BaseURL,
		model:         cfg.Model,
		apiKey:        cfg.APIKey,
		timeout:       cfg.Timeout,
		streamTimeout: cfg.StreamTimeout,
	}
}
