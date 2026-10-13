package app

import (
	"context"
	openai "delivery-ai-assistant/internal/clients/open_ai"
	"delivery-ai-assistant/internal/config"
	"delivery-ai-assistant/internal/service"
	"delivery-ai-assistant/internal/transport/api"
	"delivery-ai-assistant/pkg/logger"
	"fmt"
	"net/http"
	"time"
)

type App struct {
	server *http.Server
}

func New(ctx context.Context) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger.Init(cfg.Service.Name, cfg.Service.Env)

	openAIClient := openai.New(NewStreamingClient(), cfg.LLM)

	svc := service.New(openAIClient)
	handlers := api.NewChatHandler(svc)

	return &App{
		server: initServer(ctx, handlers, cfg.HTTPServer.Port),
	}, nil
}

func (app *App) Start() error {
	logger.Info(fmt.Sprintf("listening on %s", app.server.Addr))
	if err := app.server.ListenAndServe(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	return nil
}

func NewStreamingClient() *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}

	return &http.Client{
		Transport: transport,
	}
}
