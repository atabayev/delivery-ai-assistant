package controller

import (
	"context"
	"fmt"
)

func (ctrl *Controller) Stream(ctx context.Context, msg string) error {
	err := ctrl.svc.StreamMessage(ctx, msg)
	if err != nil {
		return fmt.Errorf("ctrl.svc.SendMessage: %w", err)
	}

	return nil
}
