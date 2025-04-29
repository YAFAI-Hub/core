package executors

import (
	"context"
	"yafai/internal/bridge/skill"
	"yafai/internal/nexus/providers"
)

type ILLMActor interface {
	GetInfo() (name string, description string)
	SetupPrompt() (prompt string, err error)
	UpdatePrompt() error
	Execute(ctx context.Context, req *YafaiRequest) (res *YafaiResponse, err error)
	Parse() error
}

type YafaiAgent struct {
	Name          string                  `yaml:"-"`
	Description   string                  `yaml:"description"`
	Capabilities  string                  `yaml:"capabilities,omitempty"`
	Model         string                  `yaml:"model"`
	Provider      string                  `yaml:"provider"`
	GenAIProvider providers.GenAIProvider `yaml:"provider_obj,omitempty"`
	Goal          string                  `yaml:"goal"`
	DependsOn     string                  `yaml:"depends"`
	RespondsTo    string                  `yaml:"responds"`
	SysPrompt     string                  `yaml:"sys_prompt,omitempty"`
	Actions       []*skill.Action
	Tools         []providers.LLMTool `yaml:"tools,omitempty"`
	History       []*ChatRecord       `json:"history,omitempty"`
	//Integrations  map[string]interface{}   `yaml:"integrations"`
	SkillClient skill.SkillServiceClient
	Status      string            `yaml:"status"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
}

type YafaiOrchestrator struct {
	Name          string                  `json:"name"`
	Description   string                  `json:"description"`
	Scope         string                  `json:"scope"`
	Goal          string                  `json:"goal"`
	Model         string                  `json:"model"`
	Provider      string                  `json:"provider"`
	GenAIProvider providers.GenAIProvider `json:"provider_obj"`
	Team          map[string]*YafaiAgent  `json:"team"`
	SysPrompt     string                  `json:"prompt,omitempty"`
	History       []*ChatRecord           `json:"history,omitempty"`
	Plan          *PlannerResponse        `json:"plan,omitempty"`
	PlanConfirmed bool                    `json:"plan_confirmed"`
}

type YafaiPlanner struct {
	Agents        []*YafaiAgent           `yaml:"agents,omitempty"`
	Model         string                  `yaml:"model"`
	Provider      string                  `yaml:"provider,omitempty"`
	GenAIProvider providers.GenAIProvider `yaml:"provider_obj,omitempty"`
	Tasks         []*PlannerTask          `yaml:"tasks,omitempty"`
	SysPrompt     string                  `yaml:"sys_prompt,omitempty"`
}

type YafaiRequest struct {
	Source  string
	Request *providers.RequestMessage
}

type YafaiResponse struct {
	Source   string
	Response *providers.ResponseMessage
}

type ChatRecord struct {
	From    string
	To      string
	Message string
}

type OrchestratorPromptStruct struct {
	Agents       string
	ChatRecords  string
	Confirmation string
	Scope        string
}

// Planner Types
type PlannerRequest struct {
	Request string
}

type PlannerTask struct {
	Task      string
	Agent     string
	Thought   string
	DependsOn string
}

type PlannerResponse struct {
	Response []*PlannerTask
}

type PlannerTemplateStruct struct {
	Agents string
}

//Agent Types

//Orchestrator Types

type AgentDescription struct {
	Name         string
	Description  string
	Capabilities string
}

type ToolDescription struct {
	Name        string
	Description string
}

var AgentReactStep struct {
	Thought     string               `json:"thought"`
	Action      string               `json:"action"`
	Input       interface{}          `json:"input"`
	FinalAnswer string               `json:"final_answer"`
	ToolCalls   []providers.ToolCall `json:"tool_calls"`
}

type AgentTemplateStruct struct {
	Tools       string
	ChatHistory string
	Scratchpad  string
}

type OrchReactStep struct {
	Thought     string
	InvokeAgent string
	AgentInput  string
	Observation string
}

type AgentLogs struct {
	AgentLogs string
}

type Param struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	In          string `json:"in"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type Action struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Method      string            `json:"method"`
	BaseURL     string            `json:"baseUrl"`
	Path        string            `json:"path"`
	Headers     map[string]string `json:"headers"`
	Params      []Param           `json:"params"`
}

type ToolExecutionInput struct {
	Name        string                 `json:"name"`
	QueryParams map[string]interface{} `json:"queryParams"`
	PathParams  map[string]interface{} `json:"pathParams"`
	BodyParams  map[string]interface{} `json:"bodyParams"`
}
