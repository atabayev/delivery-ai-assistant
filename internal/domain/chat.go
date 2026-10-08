package domain

type ChatReply struct {
	Message string
	Model   string
	Token   Tokens
}

type Tokens struct {
	Prompt     int
	Completion int
	Total      int
}

type StreamResult struct {
	Text string
	Err  error
}
