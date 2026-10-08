package controller

import (
	"context"
	"delivery-ai-assistant/internal/domain"
)

type Service interface {
	SendMessage(context.Context, string) (domain.ChatReply, error)
	StreamMessage(context.Context, string) error
}

type Controller struct {
	svc Service
}

func New(svc Service) *Controller {
	return &Controller{
		svc: svc,
	}
}
