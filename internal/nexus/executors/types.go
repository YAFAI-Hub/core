package executors

import (
	"context"
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
	Name          string                   `yaml:"name"`
	Description   string                   `yaml:"description"`
	Capabilities  string                   `yaml:"capabilities,omitempty"`
	Model         string                   `yaml:"model"`
	Provider      string                   `yaml:"provider"`
	GenAIProvider providers.GenAIProvider  `yaml:"provider_obj,omitempty"`
	Goal          string                   `yaml:"goal"`
	DependsOn     string                   `yaml:"depends"`
	RespondsTo    string                   `yaml:"responds"`
	SysPrompt     string                   `yaml:"sys_prompt,omitempty"`
	Tools         []providers.ToolFunction `yaml:"tools,omitempty"`
	//Integrations  map[string]interface{}   `yaml:"integrations"`
	Status   string            `yaml:"status"`
	Metadata map[string]string `yaml:"metadata,omitempty"`
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

type ToolsStruct struct {
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

type AgentReactStep struct {
	Thought     string
	Action      string
	Input       string
	Observation string
}

type AgentLogs struct {
	AgentLogs string
}
