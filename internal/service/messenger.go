package service

import (
	"context"
	"delivery-ai-assistant/internal/domain"
)

type Messenger interface {
	Chat(context.Context, string) (domain.ChatReply, error)
	StreamChat(context.Context, string) (<-chan domain.StreamResult, error)
}
