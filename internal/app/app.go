package app

import (
	"context"
	openai "delivery-ai-assistant/internal/clients/open_ai"
	"delivery-ai-assistant/internal/config"
	"delivery-ai-assistant/internal/service"
	"delivery-ai-assistant/internal/transport/api"
	"fmt"
	"log"
	"net/http"
)

type App struct {
	server *http.Server
}

func New(ctx context.Context) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	openAIClient := openai.New(
		&http.Client{},
		cfg.OpenAI.BaseURL,
		cfg.OpenAI.Model,
		cfg.OpenAI.APIKey,
		cfg.OpenAI.Timeout,
		cfg.OpenAI.StreamTimeout,
	)

	svc := service.New(openAIClient)
	handlers := api.NewChatHandler(svc)

	return &App{
		server: initServer(ctx, handlers, cfg.HTTPServer.Port),
	}, nil
}

func (app *App) Start() error {
	log.Println("listening ", app.server.Addr)

	if err := app.server.ListenAndServe(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	return nil
}
