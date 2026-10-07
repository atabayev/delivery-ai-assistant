package openai

import (
	"bytes"
	"context"
	"delivery-ai-assistant/internal/domain"
	"encoding/json"
	"fmt"
	"net/http"
)

const pathSend = "/chat/completions"

type requestBody struct {
	Model       string       `json:"model"`
	Messages    []reqMessage `json:"messages"`
	Temperature *float64     `json:"temperature,omitempty"`
}

type reqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseBody struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type errResponseBody struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Send sends a user message to the OpenAI Chat Completions API
// and returns the model reply along with token usage.
func (cli *Client) Send(ctx context.Context, msg string) (domain.ChatReply, error) {
	req, err := cli.prepareRequest(ctx, msg)
	if err != nil {
		return domain.ChatReply{}, fmt.Errorf("cannot prepare request: %w", err)
	}

	resp, err := cli.httpClient.Do(req)
	if err != nil {
		return domain.ChatReply{}, fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	return cli.handleResponse(resp)
}

func (cli *Client) prepareRequest(ctx context.Context, msg string) (*http.Request, error) {
	fullURL := cli.baseURL + pathSend

	body, err := cli.prepareRequestBody(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Authorization", "Bearer "+cli.apiKey)

	return req, nil
}

func (cli *Client) prepareRequestBody(msg string) ([]byte, error) {
	reqBody := requestBody{
		Model: cli.model,
		Messages: []reqMessage{
			{Role: "user", Content: msg},
		},
	}

	return json.Marshal(reqBody)
}

func (cli *Client) handleResponse(resp *http.Response) (domain.ChatReply, error) {
	if err := cli.checkStatusCode(resp); err != nil {
		return domain.ChatReply{}, err
	}

	var respBody responseBody

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return domain.ChatReply{}, fmt.Errorf("failed to decode response body: %w", err)
	}

	if len(respBody.Choices) == 0 {
		return domain.ChatReply{}, fmt.Errorf("empty choices in response")
	}

	return domain.ChatReply{
		Message: respBody.Choices[0].Message.Content,
		Model:   respBody.Model,
		Token: domain.Tokens{
			Prompt:     respBody.Usage.PromptTokens,
			Completion: respBody.Usage.CompletionTokens,
			Total:      respBody.Usage.TotalTokens,
		},
	}, nil
}

func (cli *Client) checkStatusCode(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var errResp errResponseBody

	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return fmt.Errorf("failed to read error response body, status_code: %d: %w", resp.StatusCode, err)
	}

	return fmt.Errorf("cannot send request, status_code: %d: %s", resp.StatusCode, errResp.Error.Message)
}
