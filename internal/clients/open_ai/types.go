package openai

type requestBody struct {
	Model         string         `json:"model"`
	Messages      []reqMessage   `json:"messages"`
	Stream        *bool          `json:"stream,omitempty"`
	StreamOptions *streamOptions `json:"stream_options,omitempty"`
	Temperature   *float64       `json:"temperature,omitempty"`
}

type streamOptions struct {
	IncludeUsage *bool `json:"include_usage,omitempty"`
}

type reqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseBody struct {
	Model   string `json:"model"`
	Choices []struct {
		Message      *messageContent `json:"message,omitempty"`
		Delta        *messageContent `json:"delta,omitempty"`
		FinishReason *string         `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage *usage `json:"usage"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type messageContent struct {
	Content   string  `json:"content"`
	Reasoning *string `json:"reasoning"`
}

type errResponseBody struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
