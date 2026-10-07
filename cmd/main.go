package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	openai "delivery-ai-assistant/internal/clients/open_ai"
	"delivery-ai-assistant/internal/config"
	"delivery-ai-assistant/internal/controller"
	"delivery-ai-assistant/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	start := time.Now()

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd <message>")
		return
	}

	var message strings.Builder

	for _, arg := range os.Args[1:] {
		message.WriteString(" " + arg)
	}

	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	httpClient := &http.Client{
		Timeout: cfg.HTTP.Timeout,
	}

	openAIClient := openai.New(
		httpClient,
		cfg.OpenAI.BaseURL,
		cfg.OpenAI.Model,
		cfg.OpenAI.APIKey,
	)

	svc := service.New(openAIClient)
	ctrl := controller.New(svc)

	resp, err := ctrl.Send(ctx, message.String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "send: %v\n", err)
		os.Exit(1)
	}

	finish := time.Since(start)

	fmt.Println(resp.Message)
	fmt.Println()
	fmt.Printf("model=%s prompt_tokens=%d completion_tokens=%d latency=%.1fs\n",
		cfg.OpenAI.Model, resp.Token.Prompt, resp.Token.Completion, finish.Seconds())
}
