package executors

import (
	//"fmt"
	"bytes"
	"context"
	"log/slog"
	"text/template"

	"yafai/internal/nexus/assets/templates"
	"yafai/internal/nexus/providers"
)

func (a *YafaiAgent) GetInfo() (name string, description string) {
	// Implement the initialization logic for the agent
	return a.Name, a.Description
}

func (a *YafaiAgent) SetupPrompt() (prompt string, err error) {
	// Implement the logic to set up the initial system prompt for the agent
	// var tool_desc string

	// for _, tool := range a.Tools {
	// 	var output bytes.Buffer
	// 	tmpl, err := template.New("teamDescription").Parse(templates.TeamDescriptionTemplate)
	// 	if err != nil {
	// 		slog.Error(err.Error())
	// 	}
	// 	var data = ToolDescription{Name: tool.Name, Description: tool.Description}
	// 	err = tmpl.Execute(&output, data)
	// 	if err != nil {
	// 		slog.Error(err.Error())
	// 	}

	// 	tool_desc += output.String()
	// }

	system_tmpl, err := template.New("AgentSystem").Parse(templates.AgentTemplate)
	if err != nil {
		slog.Error(err.Error())
	}

	var inst_data = ToolsStruct{Tools: "", ChatHistory: "", Scratchpad: ""}
	var system_prompt_string bytes.Buffer

	if err != nil {
		slog.Error(err.Error())
	}

	err = system_tmpl.Execute(&system_prompt_string, inst_data)
	if err != nil {
		slog.Error(err.Error())
	}
	slog.Info(system_prompt_string.String())
	return system_prompt_string.String(), err
}

func (a *YafaiAgent) UpdatePrompt() error {
	// Implement the logic to update the system prompt based on the converstations state t-3 conversations history + t-3 react steps.
	return nil
}

func (a *YafaiAgent) Execute(ctx context.Context, req *YafaiRequest) (res *YafaiResponse, err error) {
	// Implement the logic to execute the agent's task
	// sys_prompt, err := a.SetupPrompt()
	// if err != nil {
	// 	slog.Error(err.Error())
	// }
	provider := providers.GetProvider(a.Provider)
	client := provider.Init()
	YafaiRequest := providers.RequestMessage{Role: "user", Content: req.Request.Content}
	systemRequest := providers.RequestMessage{Role: "system", Content: a.SysPrompt}
	provider_req := providers.GenAIProviderRequest{Model: a.Model, Messages: []providers.RequestMessage{YafaiRequest, systemRequest}, Stream: false}
	response, err := provider.Generate(ctx, client, provider_req)
	if err != nil {
		slog.Error(err.Error())
	}
	provider.Close(client)
	//slog.Info("here is the raw response--->%v", response)
	return &YafaiResponse{Response: &response.Choices[0].Message}, err
}

func (a *YafaiAgent) Parse() error {
	// Implement the logic to parse the agent's response
	return nil
}
