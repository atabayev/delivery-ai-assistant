package service

import (
	"context"
	"delivery-ai-assistant/internal/domain"
	"fmt"
)

func (svc *Service) StreamChat(ctx context.Context, msg domain.ChatMessage) (<-chan domain.StreamResult, error) {
	chStream, err := svc.messenger.StreamChat(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("svc.messenger.Send: %w", err)
	}

	return chStream, nil
}
