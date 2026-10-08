package service

import (
	"context"
	"delivery-ai-assistant/internal/domain"
	"fmt"
)

func (svc *Service) SendMessage(ctx context.Context, msg string) (domain.ChatReply, error) {
	resp, err := svc.messenger.Chat(ctx, msg)
	if err != nil {
		return domain.ChatReply{}, fmt.Errorf("svc.messenger.Send: %w", err)
	}

	return resp, nil
}
