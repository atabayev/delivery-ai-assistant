package domain

import "time"

type ChatReply struct {
	Content string
	Model   string
	Usage   Usage
	Latency time.Duration
}

type Usage struct {
	Prompt     int
	Completion int
	Total      int
}

// StreamKind distinguishes a text chunk from the terminal stream events.
type StreamKind int

const (
	StreamDelta StreamKind = iota
	StreamDone
	StreamError
)

const (
	StreamCodeUpstream   = "upstream_error"
	StreamMsgInterrupted = "stream interrupted"
)

type StreamResult struct {
	Kind    StreamKind
	Text    string
	Usage   Usage
	TTFT    time.Duration
	Total   time.Duration
	Err     error
	ErrCode string
	ErrMsg  string
}

// ChatRequest описывает входящий запрос для чата
type ChatMessage struct {
	Messages []Message `json:"messages"`
	Model    string    `json:"model"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
