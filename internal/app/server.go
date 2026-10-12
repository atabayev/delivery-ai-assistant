package app

import (
	"context"
	"delivery-ai-assistant/internal/transport/api"
	"fmt"
	"net/http"
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
