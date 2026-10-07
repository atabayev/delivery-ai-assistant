package openai

import "net/http"

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
}

func New(
	httpClient *http.Client,
	baseURL string,
	model string,
	apiKey string,
) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		model:      model,
		apiKey:     apiKey,
	}
}
