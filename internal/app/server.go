package app

import (
	"context"
	"fmt"
	"net/http"

	"delivery-ai-assistant/internal/transport/api"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func initServer(ctx context.Context, handlers *api.ChatHandler, port int) *http.Server {
	mux := http.NewServeMux()

	chatHandler := withAppContext(ctx)(
		limitBodySize(1024 * 1024)(
			http.HandlerFunc(handlers.Chat),
		),
	)

	mux.Handle("POST /api/chat", chatHandler)
	mux.HandleFunc("GET /healthz", handlers.HealthZ)
	mux.Handle("/metrics", promhttp.Handler())

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
}

func limitBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

func withAppContext(ctx context.Context) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			go func() {
				select {
				case <-r.Context().Done():
					cancel() // клиент ушел раньше времени
				case <-ctx.Done():
					// приложение завершилось (SIGTERM)
				}
			}()

			// Передаем запрос с новым объединенным контекстом дальше
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
