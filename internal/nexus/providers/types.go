package providers

//import "nexus/providers"

type RequestMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Image   []string `json:"image,omitempty"`
	//Tools   []providers.Tool `json:"tool,omitempty"`
}

type LLMTool struct {
	Type     string      `json:"type"`
	Function LLMFunction `json:"function"`
}

type LLMFunction struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Parameters  LLMFunctionParameters `json:"parameters"`
}

type LLMFunctionParameters struct {
	Type       string                 `json:"type"` // always "object"
	Properties map[string]LLMProperty `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

type LLMProperty struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Enum        []string               `json:"enum,omitempty"`
	Properties  map[string]LLMProperty `json:"properties,omitempty"` // for objects
	Required    []string               `json:"required,omitempty"`   // for objects
	Items       *LLMProperty           `json:"items,omitempty"`      // for arrays
}

type ToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolCallFunc `json:"function"`
}

type ResponseMessage struct {
	Role      string     `json:"role"`
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
	Model           string           `json:"model"`
	Messages        []RequestMessage `json:"messages"`
	ResponseFormat  interface{}      `json:"response_format,omitempty"`
	ReasoningFormat string           `json:"reasoning_format,omitempty"`
	Stream          bool             `json:"stream"`
	Tools           []LLMTool        `json:"tools"`
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

type ResponseFormat struct {
	Type string `json:"type"`
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
