package main

import (
	"context"
	"delivery-ai-assistant/internal/app"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	deliveryApp, err := app.New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	err = deliveryApp.Start()
	if err != nil {
		log.Fatal(err)
	}
}
