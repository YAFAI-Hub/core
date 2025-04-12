package executors

import (
	//"fmt"
	"bytes"
	"context"
	"log/slog"
	"strings"
	"text/template"

	"yafai/internal/nexus/assets/templates"
	"yafai/internal/nexus/providers"
)

func (o *YafaiOrchestrator) SetupPrompt() (prompt string, err error) {

	system_tmpl, err := template.New("OrchSystem").Parse(templates.OrchestratorPrompt)
	if err != nil {
		slog.Error(err.Error())
	}
	chats, err := o.getChatHistory()
	if err != nil {
		slog.Error(err.Error())
	}
	var orch_data = OrchestratorPromptStruct{ChatRecords: chats, Confirmation: "not confirmed", Scope: o.Scope}

	var system_prompt_string bytes.Buffer

	err = system_tmpl.Execute(&system_prompt_string, orch_data)
	if err != nil {
		slog.Error(err.Error())
	}
	slog.Info(system_prompt_string.String())
	return system_prompt_string.String(), err

}

func (o *YafaiOrchestrator) GetInfo() (name string, description string) {
	// Implement the initialization logic for the agent
	return o.Name, o.Description
}

func (o *YafaiOrchestrator) getChatHistory() (chats string, err error) {

	var historyBuilder strings.Builder
	for _, record := range o.History {
		historyBuilder.WriteString("From: " + record.From + "\n")
		historyBuilder.WriteString("To: " + record.To + "\n")
		historyBuilder.WriteString("Message: " + record.Message + "\n")
		historyBuilder.WriteString("-----\n")
	}
	chats = historyBuilder.String()
	return chats, nil
}
func (o *YafaiOrchestrator) AppendChatRecord(From string, To string, Message string) error {
	// Implement the logicto append a new chat record to the conversation history
	record := &ChatRecord{From: From, To: To, Message: Message}
	o.History = append(o.History, record)
	return nil
}

func (o *YafaiOrchestrator) UpdatePlan(plan *PlannerResponse) error {
	o.Plan = plan
	o.PlanConfirmed = false
	return nil
}

func (o *YafaiOrchestrator) UpdatePlanStatus(confirm bool) error {
	o.PlanConfirmed = confirm
	return nil
}

func (a *YafaiOrchestrator) UpdatePrompt() error {
	// Implement the logic to update the system prompt based on the converstations state t-3 conversations history + t-3 react steps.
	return nil
}

func (o *YafaiOrchestrator) AttachTeam() error {
	return nil
}

func (o *YafaiOrchestrator) Execute(ctx context.Context, req *YafaiRequest) (res *YafaiResponse, err error) {
	// Implement the logic to execute the agent's task
	sys_prompt, err := o.SetupPrompt()
	if err != nil {
		slog.Error(err.Error())
	}
	provider := providers.GetProvider(o.Provider)
	client := provider.Init()
	system_request := providers.RequestMessage{Role: "system", Content: sys_prompt}
	user_request := providers.RequestMessage{Role: "user", Content: req.Request.Content}

	provider_req := providers.GenAIProviderRequest{Model: o.Model, Messages: []providers.RequestMessage{system_request, user_request}, Stream: false}
	completion, err := provider.Generate(ctx, client, provider_req)
	payload := &providers.ResponseMessage{Role: "assistant", Content: completion.Choices[0].Message.Content}
	payload.Content = strings.ReplaceAll(payload.Content, "\\", "")
	//payload.Content = strings.ReplaceAll(strings.ReplaceAll(payload.Content, "\n", ""), "\\", "")
	return &YafaiResponse{Source: "orchestrator", Response: payload}, err
}

func (o *YafaiOrchestrator) Parse(ctx context.Context, agent_log map[string]string) (response string, err error) {
	// Implement the logic to prepare orchestrator based on agent logs.
	synth_tmpl, err := template.New("OrchSynth").Parse(templates.SynthPrompt)
	if err != nil {
		slog.Error(err.Error())
	}
	var synth_string bytes.Buffer
	var builder strings.Builder
	for name, log := range agent_log {
		builder.WriteString("Name: " + name + "\n")
		builder.WriteString("Output: " + log + "\n")
	}

	logs := AgentLogs{AgentLogs: builder.String()}
	err = synth_tmpl.Execute(&synth_string, logs)
	if err != nil {
		slog.Error(err.Error())
	}

	//synth_prompt, err :=
	if err != nil {
		slog.Error(err.Error())
	}
	provider := providers.GetProvider(o.Provider)
	client := provider.Init()
	system_request := providers.RequestMessage{Role: "system", Content: synth_string.String()}
	user_request := providers.RequestMessage{Role: "user", Content: "analyse the agent logs and give final answer"}

	provider_req := providers.GenAIProviderRequest{Model: o.Model, Messages: []providers.RequestMessage{system_request, user_request}, Stream: false}
	completion, err := provider.Generate(ctx, client, provider_req)
	if err != nil {
		slog.Error(err.Error())
	}
	payload := &providers.ResponseMessage{Role: "assistant", Content: completion.Choices[0].Message.Content}
	payload.Content = strings.ReplaceAll(payload.Content, "\\", "")
	return payload.Content, nil
}
