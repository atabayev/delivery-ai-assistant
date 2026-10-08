package openai

import (
	"bufio"
	"context"
	"delivery-ai-assistant/internal/domain"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

var errContextDone = errors.New("context done")

// StreamSend sends a user message to the OpenAI Chat Completions API
// and returns the model reply along with token usage.
func (cli *Client) StreamChat(ctx context.Context, msg string) (<-chan domain.StreamResult, error) {
	req, err := cli.prepareRequest(ctx, msg, true)
	if err != nil {
		return nil, fmt.Errorf("cannot prepare request: %w", err)
	}

	resp, err := cli.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if err = cli.checkStatusCode(resp); err != nil {
		return nil, err
	}

	return cli.handleStreamResponse(ctx, resp)
}

func (cli *Client) handleStreamResponse(
	ctx context.Context,
	resp *http.Response,
) (<-chan domain.StreamResult, error) {
	// Do not communicate by sharing memory; instead, share memory by communicating! (c) Rob Pike
	msgChan := make(chan domain.StreamResult)

	go func(ctx context.Context) {
		defer resp.Body.Close()
		defer close(msgChan)

		start := time.Now()

		var ttft time.Duration

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
				cli.sendMsg(ctx, msgChan, domain.StreamResult{
					Err: fmt.Errorf("cannot unmarshal stream body: %w", err),
				})
				continue
			}

			if streamBody.Usage != nil {
				cli.sendMsg(ctx, msgChan, domain.StreamResult{
					Text: cli.prepareUsageData(streamBody.Usage, ttft, time.Since(start)),
				})
				break
			}

			if len(streamBody.Choices) == 0 {
				continue
			}

			choice := streamBody.Choices[0]

			if streamBody.Choices[0].Delta.Content != "" {
				if ttft == 0 {
					ttft = time.Since(start)
				}

				cli.sendMsg(ctx, msgChan, domain.StreamResult{
					Text: choice.Delta.Content,
				})
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case msgChan <- domain.StreamResult{Err: err}:
			case <-ctx.Done():
			}
		}
	}(ctx)

	return msgChan, nil
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
