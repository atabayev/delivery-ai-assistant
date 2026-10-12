package service

import (
	"context"
	"delivery-ai-assistant/internal/domain"
)

type Messenger interface {
	Chat(context.Context, domain.ChatMessage) (domain.ChatReply, error)
	StreamChat(context.Context, domain.ChatMessage) (<-chan domain.StreamResult, error)
}
