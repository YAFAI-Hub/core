package providers

//import "nexus/providers"

type Role string

type RequestMessage struct {
	Role    Role     `json:"role"`
	Content string   `json:"content"`
	Image   []string `json:"image,omitempty"`
	//Tools   []providers.Tool `json:"tool,omitempty"`
}

type ToolFunction struct {
	Name        string `json:"name"`
	Description string `jdon:"description"`
	Arguments   string `json:"arguments"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ResponseMessage struct {
	Role      Role       `json:"role"`
	Content   string     `json:"content"`
	Thought   string     `json:"reasoning,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type ResponseChoice struct {
	Index        int `json:"index"`
	Message      ResponseMessage
	Logprobs     interface{} `json:"logprobs"` 
	FinishReason string      `json:"finish_reason"`
}

type ResponseUsage struct {
	QueueTime        float64 `json:"queue_time"`
	PromptTokens     int     `json:"prompt_tokens"`
	PromptTime       float64 `json:"prompt_time"`
	CompletionTokens int     `json:"completion_tokens"`
	CompletionTime   float64 `json:"completion_time"`
	TotalTokens      int     `json:"total_tokens"`
	TotalTime        float64 `json:"total_time"`
}

type GenAIProviderRequest struct {
	Model    string           `json:"model"`
	Messages []RequestMessage `json:"messages"`
	ResponseFormat  interface{}    `json:"response_format,omitempty"`
	ReasoningFormat string         `json:"reasoning_format,omitempty"`
	Stream          bool           `json:"stream"`
	Tools           []ToolFunction `json:"tools"`
}

type GenAIProviderResponse struct {
	ID                string           `json:"id"`
	Object            string           `json:"object"`
	Created           int64            `json:"created"`
	Model             string           `json:"model"`
	Choices           []ResponseChoice `json:"choices"`
	Usage             ResponseUsage    `json:"usage"`
	SystemFingerprint string           `json:"system_fingerprint"`
}

// type GenAIProviderResponse struct {
// 	Model     string    `json:"model"`
// 	CreatedAt string    `json:"created_at"`
// 	Response  string    `json:"response"`
// 	Done      bool      `json:"done"`
// 	Context   []int     `json:"context"`
// 	TotalDuration int64 `json:"total_duration"`
// 	LoadDuration int64 `json:"load_duration"`
// 	PromptEvalCount int64 `json:"prompt_eval_count"`
// 	PromptEvalDuration int64 `json:"prompt_eval_duration"`
// 	EvalCount int64 `json:"eval_count"`
// 	EvalDuration int64 `json:"eval_duration"`
// }

// TODO:Support Streaming for Ollama
