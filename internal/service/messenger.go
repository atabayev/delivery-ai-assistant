package service

import (
	"context"
	"delivery-ai-assistant/internal/domain"
)

type Messenger interface {
	Send(context.Context, string) (domain.ChatReply, error)
}
