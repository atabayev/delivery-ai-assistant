package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	ModeSync   = "sync"
	ModeStream = "stream"

	statusOK    = "ok"
	statusError = "error"

	tokenPrompt     = "prompt"
	tokenCompletion = "completion"
)

var (
	LLMRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_requests_total",
			Help: "Total number of LLM requests.",
		},
		[]string{"model", "mode", "status"}, // mode: stream|sync, status: ok|error
	)

	LLMRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_request_duration_seconds",
			Help:    "Total duration of LLM requests in seconds.",
			Buckets: []float64{0.5, 1, 2, 5, 10, 20, 30, 60, 120},
		},
		[]string{"model", "mode"},
	)

	LLMTimeToFirstToken = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_time_to_first_token_seconds",
			Help:    "Time from request start to first streamed chunk.",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 20},
		},
		[]string{"model"},
	)

	LLMTokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_tokens_total",
			Help: "Total number of tokens consumed.",
		},
		[]string{"model", "type"}, // type: prompt|completion
	)

	LLMRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "llm_requests_in_flight",
			Help: "Number of LLM requests currently being processed.",
		},
	)
)

// Call tracks one LLM request from start until Finish.
type Call struct {
	model    string
	mode     string
	start    time.Time
	finish   sync.Once
	firstTok sync.Once
}

// Begin starts an LLM request: in-flight gauge goes up until Finish.
func Begin(model, mode string) *Call {
	LLMRequestsInFlight.Inc()

	return &Call{
		model: model,
		mode:  mode,
		start: time.Now(),
	}
}

// ObserveFirstToken records time from Begin to the first streamed fragment.
func (c *Call) ObserveFirstToken() {
	c.firstTok.Do(func() {
		LLMTimeToFirstToken.WithLabelValues(c.model).Observe(time.Since(c.start).Seconds())
	})
}

// AddTokens records prompt and completion tokens for a successful response.
func (c *Call) AddTokens(prompt, completion int) {
	if prompt > 0 {
		LLMTokensTotal.WithLabelValues(c.model, tokenPrompt).Add(float64(prompt))
	}

	if completion > 0 {
		LLMTokensTotal.WithLabelValues(c.model, tokenCompletion).Add(float64(completion))
	}
}

// Finish records request count and duration, then drops the in-flight gauge.
// err == nil is status "ok". Safe to call once; further calls are ignored.
func (c *Call) Finish(err error) {
	c.finish.Do(func() {
		status := statusOK
		if err != nil {
			status = statusError
		}

		LLMRequestsTotal.WithLabelValues(c.model, c.mode, status).Inc()
		LLMRequestDuration.WithLabelValues(c.model, c.mode).Observe(time.Since(c.start).Seconds())
		LLMRequestsInFlight.Dec()
	})
}
