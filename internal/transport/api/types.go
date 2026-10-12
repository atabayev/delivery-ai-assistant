package api

type chatRequest struct {
	Messages []message `json:"messages"`
	Model    string    `json:"model"`
	Stream   bool      `json:"stream"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Content   string `json:"content"`
	Model     string `json:"model"`
	Usage     usage  `json:"usage"`
	LatencyMs int    `json:"latency_ms"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	Total            int `json:"total"`
}

type errResponse struct {
	Error errData `json:"error"`
}

type errData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newErrResponse(code, message string) *errResponse {
	return &errResponse{
		Error: errData{
			Code:    code,
			Message: message,
		},
	}
}
