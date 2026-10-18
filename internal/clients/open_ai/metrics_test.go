package openai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"delivery-ai-assistant/internal/config"
	"delivery-ai-assistant/internal/domain"
	"delivery-ai-assistant/internal/metrics"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestChatMetricsSyncSuccess(t *testing.T) {
	const model = "sync-model"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"model":"sync-model",
			"choices":[{"message":{"role":"assistant","content":"hi"}}],
			"usage":{"prompt_tokens":3,"completion_tokens":4,"total_tokens":7}
		}`)
	}))
	t.Cleanup(srv.Close)

	beforeReq := counterValue(t, metrics.LLMRequestsTotal, model, metrics.ModeSync, "ok")
	beforeDur := histogramCount(t, metrics.LLMRequestDuration, model, metrics.ModeSync)
	beforeTTFT := histogramCount(t, metrics.LLMTimeToFirstToken, model)
	beforePrompt := counterValue(t, metrics.LLMTokensTotal, model, "prompt")
	beforeCompletion := counterValue(t, metrics.LLMTokensTotal, model, "completion")
	beforeFlight := gaugeValue(metrics.LLMRequestsInFlight)

	cli := testClient(srv.URL)
	reply, err := cli.Chat(context.Background(), domain.ChatMessage{
		Model:    model,
		Messages: []domain.Message{{Role: "user", Content: "ping"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if reply.Content != "hi" {
		t.Fatalf("content = %q", reply.Content)
	}

	if got := counterValue(t, metrics.LLMRequestsTotal, model, metrics.ModeSync, "ok") - beforeReq; got != 1 {
		t.Fatalf("requests ok delta = %v", got)
	}
	if got := histogramCount(t, metrics.LLMRequestDuration, model, metrics.ModeSync) - beforeDur; got != 1 {
		t.Fatalf("duration observations delta = %v", got)
	}
	if got := histogramCount(t, metrics.LLMTimeToFirstToken, model) - beforeTTFT; got != 0 {
		t.Fatalf("ttft observations delta = %v, want 0", got)
	}
	if got := counterValue(t, metrics.LLMTokensTotal, model, "prompt") - beforePrompt; got != 3 {
		t.Fatalf("prompt tokens delta = %v", got)
	}
	if got := counterValue(t, metrics.LLMTokensTotal, model, "completion") - beforeCompletion; got != 4 {
		t.Fatalf("completion tokens delta = %v", got)
	}
	if got := gaugeValue(metrics.LLMRequestsInFlight); got != beforeFlight {
		t.Fatalf("in_flight = %v, want %v", got, beforeFlight)
	}
}

func TestChatMetricsSyncError(t *testing.T) {
	const model = "sync-error-model"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":{"message":"nope"}}`)
	}))
	t.Cleanup(srv.Close)

	beforeErr := counterValue(t, metrics.LLMRequestsTotal, model, metrics.ModeSync, "error")
	beforeOK := counterValue(t, metrics.LLMRequestsTotal, model, metrics.ModeSync, "ok")
	beforePrompt := counterValue(t, metrics.LLMTokensTotal, model, "prompt")
	beforeFlight := gaugeValue(metrics.LLMRequestsInFlight)

	cli := testClient(srv.URL)
	_, err := cli.Chat(context.Background(), domain.ChatMessage{
		Model:    model,
		Messages: []domain.Message{{Role: "user", Content: "ping"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if got := counterValue(t, metrics.LLMRequestsTotal, model, metrics.ModeSync, "error") - beforeErr; got != 1 {
		t.Fatalf("requests error delta = %v", got)
	}
	if got := counterValue(t, metrics.LLMRequestsTotal, model, metrics.ModeSync, "ok") - beforeOK; got != 0 {
		t.Fatalf("requests ok delta = %v, want 0", got)
	}
	if got := counterValue(t, metrics.LLMTokensTotal, model, "prompt") - beforePrompt; got != 0 {
		t.Fatalf("prompt tokens delta = %v, want 0", got)
	}
	if got := gaugeValue(metrics.LLMRequestsInFlight); got != beforeFlight {
		t.Fatalf("in_flight = %v, want %v", got, beforeFlight)
	}
}

func TestStreamMetricsSuccess(t *testing.T) {
	const model = "stream-model"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"he\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"llo\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"usage\":{\"prompt_tokens\":2,\"completion_tokens\":5,\"total_tokens\":7}}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)

	beforeReq := counterValue(t, metrics.LLMRequestsTotal, model, metrics.ModeStream, "ok")
	beforeDur := histogramCount(t, metrics.LLMRequestDuration, model, metrics.ModeStream)
	beforeTTFT := histogramCount(t, metrics.LLMTimeToFirstToken, model)
	beforePrompt := counterValue(t, metrics.LLMTokensTotal, model, "prompt")
	beforeCompletion := counterValue(t, metrics.LLMTokensTotal, model, "completion")
	beforeFlight := gaugeValue(metrics.LLMRequestsInFlight)

	cli := testClient(srv.URL)
	ch, err := cli.StreamChat(context.Background(), domain.ChatMessage{
		Model:    model,
		Messages: []domain.Message{{Role: "user", Content: "ping"}},
	})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}

	var text string
	for item := range ch {
		if item.Err != nil {
			t.Fatalf("stream item: %v", item.Err)
		}
		text += item.Text
	}
	if text == "" {
		t.Fatal("empty stream")
	}

	if got := counterValue(t, metrics.LLMRequestsTotal, model, metrics.ModeStream, "ok") - beforeReq; got != 1 {
		t.Fatalf("requests ok delta = %v", got)
	}
	if got := histogramCount(t, metrics.LLMRequestDuration, model, metrics.ModeStream) - beforeDur; got != 1 {
		t.Fatalf("duration observations delta = %v", got)
	}
	if got := histogramCount(t, metrics.LLMTimeToFirstToken, model) - beforeTTFT; got != 1 {
		t.Fatalf("ttft observations delta = %v", got)
	}
	if got := counterValue(t, metrics.LLMTokensTotal, model, "prompt") - beforePrompt; got != 2 {
		t.Fatalf("prompt tokens delta = %v", got)
	}
	if got := counterValue(t, metrics.LLMTokensTotal, model, "completion") - beforeCompletion; got != 5 {
		t.Fatalf("completion tokens delta = %v", got)
	}
	if got := gaugeValue(metrics.LLMRequestsInFlight); got != beforeFlight {
		t.Fatalf("in_flight = %v, want %v", got, beforeFlight)
	}
}

func TestStreamMetricsHTTPError(t *testing.T) {
	const model = "stream-error-model"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `{"error":{"message":"down"}}`)
	}))
	t.Cleanup(srv.Close)

	beforeErr := counterValue(t, metrics.LLMRequestsTotal, model, metrics.ModeStream, "error")
	beforeTTFT := histogramCount(t, metrics.LLMTimeToFirstToken, model)
	beforeFlight := gaugeValue(metrics.LLMRequestsInFlight)

	cli := testClient(srv.URL)
	_, err := cli.StreamChat(context.Background(), domain.ChatMessage{
		Model:    model,
		Messages: []domain.Message{{Role: "user", Content: "ping"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if got := counterValue(t, metrics.LLMRequestsTotal, model, metrics.ModeStream, "error") - beforeErr; got != 1 {
		t.Fatalf("requests error delta = %v", got)
	}
	if got := histogramCount(t, metrics.LLMTimeToFirstToken, model) - beforeTTFT; got != 0 {
		t.Fatalf("ttft observations delta = %v, want 0", got)
	}
	if got := gaugeValue(metrics.LLMRequestsInFlight); got != beforeFlight {
		t.Fatalf("in_flight = %v, want %v", got, beforeFlight)
	}
}

func TestStreamMetricsInFlightUntilDone(t *testing.T) {
	const model = "stream-inflight-model"

	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"x\"}}]}\n\n")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-release
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)

	beforeFlight := gaugeValue(metrics.LLMRequestsInFlight)

	cli := testClient(srv.URL)
	ch, err := cli.StreamChat(context.Background(), domain.ChatMessage{
		Model:    model,
		Messages: []domain.Message{{Role: "user", Content: "ping"}},
	})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}

	select {
	case item := <-ch:
		if item.Err != nil {
			t.Fatalf("stream item: %v", item.Err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first chunk")
	}

	if got := gaugeValue(metrics.LLMRequestsInFlight) - beforeFlight; got != 1 {
		t.Fatalf("in_flight delta while streaming = %v", got)
	}

	close(release)

	for range ch {
	}

	if got := gaugeValue(metrics.LLMRequestsInFlight); got != beforeFlight {
		t.Fatalf("in_flight = %v, want %v", got, beforeFlight)
	}
}

func testClient(baseURL string) *Client {
	return New(&http.Client{}, config.LLM{
		BaseURL:       baseURL,
		Model:         "fallback-model",
		Timeout:       2 * time.Second,
		StreamTimeout: 2 * time.Second,
	})
}

func counterValue(t *testing.T, cv *prometheus.CounterVec, labels ...string) float64 {
	t.Helper()

	c, err := cv.GetMetricWithLabelValues(labels...)
	if err != nil {
		t.Fatalf("counter labels: %v", err)
	}

	return metricValue(t, c)
}

func gaugeValue(g prometheus.Gauge) float64 {
	var m dto.Metric
	if err := g.Write(&m); err != nil {
		panic(err)
	}

	return m.GetGauge().GetValue()
}

func histogramCount(t *testing.T, hv *prometheus.HistogramVec, labels ...string) uint64 {
	t.Helper()

	observer, err := hv.GetMetricWithLabelValues(labels...)
	if err != nil {
		t.Fatalf("histogram labels: %v", err)
	}

	metric, ok := observer.(prometheus.Metric)
	if !ok {
		t.Fatalf("histogram observer %T is not a metric", observer)
	}

	var m dto.Metric
	if err = metric.Write(&m); err != nil {
		t.Fatalf("write histogram: %v", err)
	}

	return m.GetHistogram().GetSampleCount()
}

func metricValue(t *testing.T, metric prometheus.Metric) float64 {
	t.Helper()

	var m dto.Metric
	if err := metric.Write(&m); err != nil {
		t.Fatalf("write metric: %v", err)
	}

	if m.Counter != nil {
		return m.Counter.GetValue()
	}
	if m.Gauge != nil {
		return m.Gauge.GetValue()
	}

	t.Fatal("metric has no counter or gauge value")
	return 0
}
