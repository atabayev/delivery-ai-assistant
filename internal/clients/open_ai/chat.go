package openai

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"delivery-ai-assistant/internal/domain"
	"delivery-ai-assistant/internal/metrics"

	"github.com/bytedance/sonic"
)

const pathSend = "/chat/completions"

// Chat sends a user message to the OpenAI Chat Completions API
// and returns the model reply along with token usage.
func (cli *Client) Chat(ctx context.Context, msg domain.ChatMessage) (domain.ChatReply, error) {
	call := metrics.Begin(cli.requestModel(&msg), metrics.ModeSync)

	var callErr error
	defer func() { call.Finish(callErr) }()

	req, err := cli.prepareRequest(ctx, &msg, false)
	if err != nil {
		callErr = err
		return domain.ChatReply{}, fmt.Errorf("cannot prepare request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, cli.timeout)
	defer cancel()

	resp, err := cli.httpClient.Do(req)
	if err != nil {
		callErr = err
		return domain.ChatReply{}, fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	reply, err := cli.handleResponse(resp)
	if err != nil {
		callErr = err
		return domain.ChatReply{}, err
	}

	call.AddTokens(reply.Usage.Prompt, reply.Usage.Completion)

	return reply, nil
}

func (cli *Client) requestModel(msg *domain.ChatMessage) string {
	if msg != nil && msg.Model != "" {
		return msg.Model
	}

	return cli.model
}

func (cli *Client) prepareRequest(ctx context.Context, msg *domain.ChatMessage, isStream bool) (*http.Request, error) {
	fullURL := cli.baseURL + pathSend

	body, err := cli.prepareRequestBody(msg, isStream)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (cli *Client) prepareRequestBody(msg *domain.ChatMessage, isStream bool) ([]byte, error) {
	model := cli.model
	if msg.Model != "" {
		model = msg.Model
	}

	reqBody := requestBody{
		Model: model,
	}

	for _, m := range msg.Messages {
		reqBody.Messages = append(reqBody.Messages, reqMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	if isStream {
		reqBody.Stream = &isStream
		reqBody.StreamOptions = &streamOptions{
			IncludeUsage: &isStream,
		}
	}

	return sonic.Marshal(reqBody)
}

func (cli *Client) handleResponse(resp *http.Response) (domain.ChatReply, error) {
	if err := cli.checkStatusCode(resp); err != nil {
		return domain.ChatReply{}, err
	}

	var respBody responseBody

	if err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return domain.ChatReply{}, fmt.Errorf("failed to decode response body: %w", err)
	}

	if len(respBody.Choices) == 0 {
		return domain.ChatReply{}, fmt.Errorf("empty choices in response")
	}

	return domain.ChatReply{
		Content: respBody.Choices[0].Message.Content,
		Model:   respBody.Model,
		Usage: domain.Usage{
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

	if err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return fmt.Errorf("failed to read error response body, status_code: %d: %w", resp.StatusCode, err)
	}

	return fmt.Errorf("cannot send request, status_code: %d: %s", resp.StatusCode, errResp.Error.Message)
}
