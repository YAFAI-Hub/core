package executors

import (
	//"fmt"
	"bytes"
	"context"
	"log/slog"
	"strings"
	"text/template"
	"time"

	"yafai/internal/nexus/assets/templates"
	"yafai/internal/nexus/providers"

	db "yafai/internal/bridge/db"
)

func (o *YafaiOrchestrator) SetupPrompt(db_conn *db.DBWrapper, thread int64) (prompt string, err error) {

	system_tmpl, err := template.New("OrchSystem").Parse(templates.OrchestratorPrompt)
	if err != nil {
		slog.Error(err.Error())
	}
	chats, err := o.getChatHistory(db_conn, thread)
	if err != nil {
		slog.Error(err.Error())
	}
	var orch_data = OrchestratorPromptStruct{Agents: o.GetAgentInfo(), ChatRecords: chats, Confirmation: "not confirmed", Scope: o.Scope}

	var system_prompt_string bytes.Buffer

	err = system_tmpl.Execute(&system_prompt_string, orch_data)
	if err != nil {
		slog.Error(err.Error())
	}
	slog.Info(system_prompt_string.String())
	return system_prompt_string.String(), err

}

func (o *YafaiOrchestrator) GetAgentInfo() (agents string) {
	var agentsBuilder strings.Builder
	for _, agent := range o.Team {

		agentsBuilder.WriteString("Name : " + agent.Name + "\n")
		agentsBuilder.WriteString("Description: " + agent.Description + "\n")
		agentsBuilder.WriteString("Capabilities: " + agent.Capabilities + "\n")
		agentsBuilder.WriteString("-----\n")
	}
	// Implement the initialization logic for the agent
	return agentsBuilder.String()
}

func (o *YafaiOrchestrator) getChatHistory(db_conn *db.DBWrapper, thread int64) (chats string, err error) {

	var historyBuilder strings.Builder

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	thread_obj, err := db_conn.GetThreadByID(ctx, thread)
	if err != nil {
		slog.Error("Fetching thread failed: %v", err.Error())
	}
	for _, record := range thread_obj.Messages {
		historyBuilder.WriteString("From: " + record.From + "\n")
		historyBuilder.WriteString("To: " + record.To + "\n")
		historyBuilder.WriteString("Message: " + record.Content + "\n")
		historyBuilder.WriteString("-----\n")
	}
	chats = historyBuilder.String()
	return chats, nil
}

func (o *YafaiOrchestrator) AppendChatRecord(db_conn *db.DBWrapper, thread int64, message db.Message) error {
	// Implement the logicto append a new chat record to the conversation history

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err := db_conn.AddMessageToThread(ctx, &message)
	if err != nil {
		slog.Error("Failed to add message to thread", "error", err)
		return err
	}
	return nil
	//o.History = append(o.History, record)
	return nil
}

// func (o *YafaiOrchestrator) UpdatePlan(plan *PlannerResponse) error {
// 	o.Plan = plan
// 	o.PlanConfirmed = false
// 	return nil
// }

// func (o *YafaiOrchestrator) UpdatePlanStatus(confirm bool) error {
// 	o.PlanConfirmed = confirm
// 	return nil
// }

func (a *YafaiOrchestrator) UpdatePrompt() error {
	// Implement the logic to update the system prompt based on the converstations state t-3 conversations history + t-3 react steps.
	return nil
}

func (o *YafaiOrchestrator) AttachTeam() error {
	return nil
}

func (o *YafaiOrchestrator) Execute(ctx context.Context, db *db.DBWrapper, thread int64, req *YafaiRequest) (res *YafaiResponse, err error) {
	// Implement the logic to execute the agent's task
	sys_prompt, err := o.SetupPrompt(db, thread)
	if err != nil {
		slog.Error(err.Error())
	}
	provider := providers.GetProvider(o.Provider)
	client := provider.Init()
	system_request := providers.RequestMessage{Role: "system", Content: sys_prompt}
	user_request := providers.RequestMessage{Role: "user", Content: req.Request.Content}

	provider_req := providers.GenAIProviderRequest{Model: o.Model, Messages: []providers.RequestMessage{system_request, user_request}, Stream: false, ResponseFormat: &providers.ResponseFormat{Type: "json_object"}}
	completion, err := provider.Generate(ctx, client, provider_req)
	if err != nil {
		slog.Error("Error generating response from provider", "error", err)
		return nil, err
	}

	if err != nil {
		slog.Error("Error generating response from orchestrator", "error", err)
		return nil, err
	}

	payload := &providers.ResponseMessage{Role: "assistant", Content: completion.Choices[0].Message.Content}
	return &YafaiResponse{Source: "orchestrator", Response: payload}, err
}
