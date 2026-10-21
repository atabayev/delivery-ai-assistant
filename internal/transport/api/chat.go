package api

import (
	"delivery-ai-assistant/internal/domain"
	"delivery-ai-assistant/pkg/logger"
	"fmt"
	"net/http"

	"github.com/bytedance/sonic"
)

func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req chatRequest

	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validateChatReq(w, req); err != nil {
		logger.Error("invalid request body", "error", err)
		return
	}

	msg := h.chatReqToDomain(req)

	if req.Stream {
		h.handleStreamChat(w, r, msg)
		return
	}

	h.handleRegularChat(w, r, msg)
}

func (h *ChatHandler) handleRegularChat(
	w http.ResponseWriter,
	r *http.Request,
	msg domain.ChatMessage,
) {
	result, err := h.svc.Chat(r.Context(), msg)
	if err != nil {
		h.handleError(w, http.StatusInternalServerError, "internal error")
		logger.Error("handling chat", "error", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err = sonic.ConfigDefault.NewEncoder(w).Encode(h.domainToChatResponse(result)); err != nil {
		logger.Error("encoding response", "error", err)
	}
}

func (h *ChatHandler) handleStreamChat(
	w http.ResponseWriter,
	r *http.Request,
	msg domain.ChatMessage,
) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.handleError(w, http.StatusInternalServerError, "streaming unsupported")
		logger.Error("streaming unsupported by response writer")
		return
	}

	streamCh, err := h.svc.StreamChat(r.Context(), msg)
	if err != nil {
		h.handleError(w, http.StatusInternalServerError, "internal error")
		logger.Error("streaming chat", "error", err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for item := range streamCh {
		switch item.Kind {
		case domain.StreamDelta:
			if item.Text == "" {
				continue
			}

			if err = writeSSE(w, flusher, "delta", streamDeltaData{Text: item.Text}); err != nil {
				logger.Error("write stream response", "error", err)
				return
			}
		case domain.StreamDone:
			err = writeSSE(w, flusher, "done", streamDoneData{
				Usage: usage{
					PromptTokens:     item.Usage.Prompt,
					CompletionTokens: item.Usage.Completion,
				},
				TTFTMs:  item.TTFT.Milliseconds(),
				TotalMs: item.Total.Milliseconds(),
			})
			if err != nil {
				logger.Error("write stream response", "error", err)
			}

			return
		case domain.StreamError:
			logger.Error("streaming chat response", "error", item.Err)

			if err = writeSSE(w, flusher, "error", streamErrorData{
				Code:    item.ErrCode,
				Message: item.ErrMsg,
			}); err != nil {
				logger.Error("write stream response", "error", err)
			}

			return
		}
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, payload any) error {
	data, err := sonic.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sse payload: %w", err)
	}

	if _, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}

	flusher.Flush()

	return nil
}

func (h *ChatHandler) chatReqToDomain(req chatRequest) domain.ChatMessage {
	res := domain.ChatMessage{
		Model:  req.Model,
		Stream: req.Stream,
	}

	for _, m := range req.Messages {
		res.Messages = append(res.Messages, domain.Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	return res
}

func (h *ChatHandler) domainToChatResponse(res domain.ChatReply) chatResponse {
	return chatResponse{
		Content: res.Content,
		Model:   res.Model,
		Usage: usage{
			PromptTokens:     res.Usage.Prompt,
			CompletionTokens: res.Usage.Completion,
		},
		LatencyMs: res.Latency.Milliseconds(),
	}
}

func (h *ChatHandler) validateChatReq(w http.ResponseWriter, req chatRequest) error {
	if len(req.Messages) == 0 {
		h.handleError(w, http.StatusBadRequest, "messages must not be empty")
		return fmt.Errorf("empty messages")
	}

	if len(req.Messages) > 50 {
		h.handleError(w, http.StatusBadRequest, "messages must not be bigger than 50")
		return fmt.Errorf("too many messages")
	}

	for _, msg := range req.Messages {
		if msg.Role != "" && msg.Role != domain.RoleSystem &&
			msg.Role != domain.RoleUser &&
			msg.Role != domain.RoleAssistant {
			h.handleError(w, http.StatusBadRequest, "invalid role")
			return fmt.Errorf("invalid role")
		}

		if msg.Content == "" {
			h.handleError(w, http.StatusBadRequest, "invalid content")
			return fmt.Errorf("empty content")
		}
	}

	return nil
}

func (h *ChatHandler) handleError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)

	codeMsg := h.mapCodeMessage(code)
	errResp := newErrResponse(codeMsg, msg)

	if err := sonic.ConfigDefault.NewEncoder(w).Encode(errResp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *ChatHandler) mapCodeMessage(code int) string {
	switch code {
	case http.StatusBadRequest:
		return "invalid_request"
	default:
		return "internal_error"
	}
}
