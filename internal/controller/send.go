package controller

import (
	"context"
	"delivery-ai-assistant/internal/domain"
	"fmt"
)

func (ctrl *Controller) Send(ctx context.Context, msg string) (domain.ChatReply, error) {
	resp, err := ctrl.svc.SendMessage(ctx, msg)
	if err != nil {
		return domain.ChatReply{}, fmt.Errorf("ctrl.svc.SendMessage: %w", err)
	}

	return resp, nil
}
