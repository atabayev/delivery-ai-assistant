package domain

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type ChatReply struct {
	Content string
	Model   string
	Usage   Usage
}

type Usage struct {
	Prompt     int
	Completion int
	Total      int
}

type StreamResult struct {
	Text string
	Err  error
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
