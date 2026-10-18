package openai

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"delivery-ai-assistant/internal/domain"
	"delivery-ai-assistant/internal/metrics"

	"github.com/bytedance/sonic"
)

// StreamChat sends a user message to the OpenAI Chat Completions API
// and returns the model reply along with token usage.
func (cli *Client) StreamChat(
	ctx context.Context,
	msg domain.ChatMessage,
) (<-chan domain.StreamResult, error) {
	call := metrics.Begin(cli.requestModel(&msg), metrics.ModeStream)

	req, err := cli.prepareRequest(ctx, &msg, true)
	if err != nil {
		call.Finish(err)
		return nil, fmt.Errorf("cannot prepare request: %w", err)
	}

	resp, err := cli.httpClient.Do(req)
	if err != nil {
		call.Finish(err)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if err = cli.checkStatusCode(resp); err != nil {
		resp.Body.Close()
		call.Finish(err)
		return nil, err
	}

	msgChan := make(chan domain.StreamResult)
	go cli.streaming(ctx, resp, msgChan, call)

	return msgChan, nil
}

func (cli *Client) streaming(
	ctx context.Context,
	resp *http.Response,
	msgChan chan domain.StreamResult,
	call *metrics.Call,
) {
	ctx, cancel := context.WithTimeout(ctx, cli.streamTimeout)
	defer cancel()

	defer resp.Body.Close()
	defer close(msgChan)

	start := time.Now()

	var (
		ttft             time.Duration
		streamErr        error
		promptTokens     int
		completionTokens int
		hasUsage         bool
	)

	// Finish before the channel is closed, so a reader that drains msgChan
	// observes the final metric values.
	defer func() {
		if streamErr == nil && hasUsage {
			call.AddTokens(promptTokens, completionTokens)
		}

		call.Finish(streamErr)
	}()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if isNeedSkip(line) {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if isNeedBreak(data) {
			break
		}

		var streamBody responseBody

		if err := sonic.Unmarshal([]byte(data), &streamBody); err != nil {
			if streamErr == nil {
				streamErr = err
			}

			cli.sendMsg(ctx, msgChan, domain.StreamResult{
				Err: fmt.Errorf("cannot unmarshal stream body: %w", err),
			})
			continue
		}

		if streamBody.Usage != nil {
			hasUsage = true
			promptTokens = streamBody.Usage.PromptTokens
			completionTokens = streamBody.Usage.CompletionTokens

			cli.sendMsg(ctx, msgChan, domain.StreamResult{
				Text: cli.prepareUsageData(streamBody.Usage, ttft, time.Since(start)),
			})
			break
		}

		if len(streamBody.Choices) == 0 {
			continue
		}

		choice := streamBody.Choices[0]
		if choice.Delta == nil || choice.Delta.Content == "" {
			continue
		}

		if ttft == 0 {
			ttft = time.Since(start)
			call.ObserveFirstToken()
		}

		cli.sendMsg(ctx, msgChan, domain.StreamResult{
			Text: choice.Delta.Content,
		})
	}

	if err := scanner.Err(); err != nil {
		if streamErr == nil {
			streamErr = err
		}

		select {
		case msgChan <- domain.StreamResult{Err: err}:
		case <-ctx.Done():
		}
	}

	if streamErr == nil && ctx.Err() != nil {
		streamErr = ctx.Err()
	}
}

func isNeedSkip(line string) bool {
	return !strings.HasPrefix(line, "data: ")
}

func isNeedBreak(data string) bool {
	return data == "[DONE]"
}

func (cli *Client) sendMsg(ctx context.Context, ch chan<- domain.StreamResult, msg domain.StreamResult) {
	select {
	case ch <- msg:
	case <-ctx.Done():
		ch <- domain.StreamResult{Err: ctx.Err()}
	}
}

func (cli *Client) prepareUsageData(usg *usage, ttft, total time.Duration) string {
	return fmt.Sprintf("\n\nmodel=%s prompt_tokens=%d completion_tokens=%d ttft=%.1fs total=%.1fs\n",
		cli.model, usg.PromptTokens, usg.CompletionTokens, ttft.Seconds(), total.Seconds())
}
