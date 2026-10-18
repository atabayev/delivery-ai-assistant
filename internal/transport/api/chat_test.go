package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"delivery-ai-assistant/internal/domain"

	"github.com/bytedance/sonic"
)

type stubService struct {
	reply     domain.ChatReply
	chatErr   error
	stream    <-chan domain.StreamResult
	streamErr error
}

func (s stubService) Chat(context.Context, domain.ChatMessage) (domain.ChatReply, error) {
	return s.reply, s.chatErr
}

func (s stubService) StreamChat(context.Context, domain.ChatMessage) (<-chan domain.StreamResult, error) {
	return s.stream, s.streamErr
}

func TestChatRejectsEmptyMessages(t *testing.T) {
	h := NewChatHandler(stubService{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"messages":[]}`))

	h.Chat(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q", got)
	}

	var body errResponse
	if err := sonic.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "invalid_request" || body.Error.Message != "messages must not be empty" {
		t.Fatalf("body = %+v", body)
	}
}

func TestChatReturnsReply(t *testing.T) {
	h := NewChatHandler(stubService{
		reply: domain.ChatReply{
			Content: "Чтобы оформить возврат, ...",
			Model:   "gpt-oss:20b",
			Usage: domain.Usage{
				Prompt:     78,
				Completion: 85,
				Total:      163,
			},
			Latency: 2300 * time.Millisecond,
		},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(
		`{"messages":[{"role":"user","content":"hi"}]}`,
	))

	h.Chat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var body chatResponse
	if err := sonic.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Content != "Чтобы оформить возврат, ..." || body.Model != "gpt-oss:20b" {
		t.Fatalf("body = %+v", body)
	}
	if body.Usage.PromptTokens != 78 || body.Usage.CompletionTokens != 85 {
		t.Fatalf("usage = %+v", body.Usage)
	}
	if body.LatencyMs != 2300 {
		t.Fatalf("latency_ms = %d", body.LatencyMs)
	}
	if strings.Contains(rec.Body.String(), `"total"`) {
		t.Fatalf("usage.total must be absent: %s", rec.Body.String())
	}
}

func TestChatInternalError(t *testing.T) {
	h := NewChatHandler(stubService{chatErr: context.Canceled})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(
		`{"messages":[{"role":"user","content":"hi"}]}`,
	))

	h.Chat(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}

	var body errResponse
	if err := sonic.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "internal_error" || body.Error.Message != "internal error" {
		t.Fatalf("body = %+v", body)
	}
}

func TestStreamChatWritesDeltaAndDone(t *testing.T) {
	ch := make(chan domain.StreamResult, 3)
	ch <- domain.StreamResult{Kind: domain.StreamDelta, Text: "Чтобы "}
	ch <- domain.StreamResult{Kind: domain.StreamDelta, Text: "оформить "}
	ch <- domain.StreamResult{
		Kind: domain.StreamDone,
		Usage: domain.Usage{
			Prompt:     78,
			Completion: 85,
		},
		TTFT:  1400 * time.Millisecond,
		Total: 2900 * time.Millisecond,
	}
	close(ch)

	h := NewChatHandler(stubService{stream: ch})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(
		`{"stream":true,"messages":[{"role":"user","content":"hi"}]}`,
	))

	h.Chat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content-type = %q", got)
	}

	events := parseSSE(t, rec.Body.String())
	if len(events) != 3 {
		t.Fatalf("events = %#v", events)
	}

	assertDelta(t, events[0], "Чтобы ")
	assertDelta(t, events[1], "оформить ")

	if events[2].Event != "done" {
		t.Fatalf("event = %q", events[2].Event)
	}
	var done streamDoneData
	if err := sonic.Unmarshal([]byte(events[2].Data), &done); err != nil {
		t.Fatalf("decode done: %v", err)
	}
	if done.Usage.PromptTokens != 78 || done.Usage.CompletionTokens != 85 {
		t.Fatalf("usage = %+v", done.Usage)
	}
	if done.TTFTMs != 1400 || done.TotalMs != 2900 {
		t.Fatalf("timing = %+v", done)
	}
}

func TestStreamChatWritesErrorWithoutDone(t *testing.T) {
	ch := make(chan domain.StreamResult, 3)
	ch <- domain.StreamResult{Kind: domain.StreamDelta, Text: "Чтобы "}
	ch <- domain.StreamResult{Kind: domain.StreamDelta, Text: "оформить "}
	ch <- domain.StreamResult{
		Kind:    domain.StreamError,
		Err:     context.Canceled,
		ErrCode: domain.StreamCodeUpstream,
		ErrMsg:  domain.StreamMsgInterrupted,
	}
	close(ch)

	h := NewChatHandler(stubService{stream: ch})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(
		`{"stream":true,"messages":[{"role":"user","content":"hi"}]}`,
	))

	h.Chat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	events := parseSSE(t, rec.Body.String())
	if len(events) != 3 {
		t.Fatalf("events = %#v", events)
	}

	assertDelta(t, events[0], "Чтобы ")
	assertDelta(t, events[1], "оформить ")

	if events[2].Event != "error" {
		t.Fatalf("event = %q", events[2].Event)
	}
	var got streamErrorData
	if err := sonic.Unmarshal([]byte(events[2].Data), &got); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if got.Code != "upstream_error" || got.Message != "stream interrupted" {
		t.Fatalf("error = %+v", got)
	}
}

func TestStreamSetupErrorIsJSON(t *testing.T) {
	h := NewChatHandler(stubService{streamErr: context.Canceled})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(
		`{"stream":true,"messages":[{"role":"user","content":"hi"}]}`,
	))

	h.Chat(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q", got)
	}

	var body errResponse
	if err := sonic.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "internal_error" || body.Error.Message != "internal error" {
		t.Fatalf("body = %+v", body)
	}
}

type sseEvent struct {
	Event string
	Data  string
}

func parseSSE(t *testing.T, body string) []sseEvent {
	t.Helper()

	if !strings.HasSuffix(body, "\n\n") {
		t.Fatalf("sse body must end with a blank line: %q", body)
	}

	var events []sseEvent
	for _, chunk := range strings.Split(body, "\n\n") {
		if chunk == "" {
			continue
		}

		var ev sseEvent
		for _, line := range strings.Split(chunk, "\n") {
			switch {
			case strings.HasPrefix(line, "event: "):
				ev.Event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				ev.Data = strings.TrimPrefix(line, "data: ")
			default:
				t.Fatalf("unexpected sse line %q", line)
			}
		}

		events = append(events, ev)
	}

	return events
}

func assertDelta(t *testing.T, ev sseEvent, text string) {
	t.Helper()

	if ev.Event != "delta" {
		t.Fatalf("event = %q", ev.Event)
	}

	var got streamDeltaData
	if err := sonic.Unmarshal([]byte(ev.Data), &got); err != nil {
		t.Fatalf("decode delta: %v", err)
	}
	if got.Text != text {
		t.Fatalf("text = %q", got.Text)
	}
}
