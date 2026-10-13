package api

import (
	"context"
	"delivery-ai-assistant/internal/domain"
)

type Service interface {
	Chat(context.Context, domain.ChatMessage) (domain.ChatReply, error)
	StreamChat(ctx context.Context, msg domain.ChatMessage) (<-chan domain.StreamResult, error)
}
type ChatHandler struct {
	svc Service
}

func NewChatHandler(svc Service) *ChatHandler {
	return &ChatHandler{svc: svc}
}
