package service

import (
	"context"
	"fmt"
)

func (svc *Service) StreamMessage(ctx context.Context, msg string) error {
	stream, err := svc.messenger.StreamChat(ctx, msg)
	if err != nil {
		return fmt.Errorf("svc.messenger.StreamChat: %w", err)
	}

	for data := range stream {
		if data.Err != nil {
			fmt.Println(data.Err)

		}

		fmt.Print(data.Text)
	}

	return nil
}
