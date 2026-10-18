package openai

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"delivery-ai-assistant/internal/domain"
	"delivery-ai-assistant/internal/metrics"

	"github.com/bytedance/sonic"
)

var errStreamInterrupted = errors.New("stream interrupted")

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

	defer func() {
		if streamErr == nil && hasUsage {
			call.AddTokens(promptTokens, completionTokens)
		}

		call.Finish(streamErr)
	}()

	fail := func(err error) {
		if streamErr == nil {
			streamErr = err
		}

		cli.sendMsg(ctx, msgChan, streamError(err))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			fail(err)
			return
		}

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
			fail(fmt.Errorf("cannot unmarshal stream body: %w", err))
			return
		}

		if streamBody.Usage != nil {
			hasUsage = true
			promptTokens = streamBody.Usage.PromptTokens
			completionTokens = streamBody.Usage.CompletionTokens

			sent := cli.sendMsg(ctx, msgChan, domain.StreamResult{
				Kind: domain.StreamDone,
				Usage: domain.Usage{
					Prompt:     promptTokens,
					Completion: completionTokens,
					Total:      streamBody.Usage.TotalTokens,
				},
				TTFT:  ttft,
				Total: time.Since(start),
			})
			if !sent && streamErr == nil {
				streamErr = ctx.Err()
			}

			return
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

		if !cli.sendMsg(ctx, msgChan, domain.StreamResult{
			Kind: domain.StreamDelta,
			Text: choice.Delta.Content,
		}) {
			if streamErr == nil {
				streamErr = ctx.Err()
			}

			return
		}
	}

	if err := scanner.Err(); err != nil {
		fail(err)
		return
	}

	if err := ctx.Err(); err != nil {
		fail(err)
		return
	}

	if !hasUsage {
		fail(errStreamInterrupted)
	}
}

func isNeedSkip(line string) bool {
	return !strings.HasPrefix(line, "data: ")
}

func isNeedBreak(data string) bool {
	return data == "[DONE]"
}

func (cli *Client) sendMsg(ctx context.Context, ch chan<- domain.StreamResult, msg domain.StreamResult) bool {
	select {
	case ch <- msg:
		return true
	case <-ctx.Done():
		ch <- streamError(ctx.Err())
		return false
	}
}

func streamError(err error) domain.StreamResult {
	return domain.StreamResult{
		Kind:    domain.StreamError,
		Err:     err,
		ErrCode: domain.StreamCodeUpstream,
		ErrMsg:  domain.StreamMsgInterrupted,
	}
}
