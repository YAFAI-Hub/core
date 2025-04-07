package executors

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"text/template"
	"yafai/internal/nexus/assets/templates"
	"yafai/internal/nexus/providers"
)

func (p *YafaiPlanner) SetupPrompt() (prompt string, err error) {
	//Implement the logic to set up the prompt for the agent
	var agent_desc string

	for _, agent := range p.Agents {
		var output bytes.Buffer
		tmpl, err := template.New("teamDescription").Parse(templates.TeamDescriptionTemplate)
		if err != nil {
			slog.Error(err.Error())
		}
		var data = AgentDescription{Name: agent.Name, Description: agent.Description}
		err = tmpl.Execute(&output, data)
		if err != nil {
			slog.Error(err.Error())
		}
		agent_desc += output.String()
	}
	var system_prompt_string bytes.Buffer
	system_tmpl, err := template.New("PlannerSystem").Parse(templates.PlannerTemplate)
	if err != nil {
		slog.Error(err.Error())
	}

	inst_data := PlannerTemplateStruct{Agents: agent_desc}
	err = system_tmpl.Execute(&system_prompt_string, inst_data)
	if err != nil {
		slog.Error(err.Error())
	}

	sys_prompt := system_prompt_string.String()

	return sys_prompt, err
}

func (p *YafaiPlanner) Execute(ctx context.Context, req *YafaiRequest) (response *YafaiResponse, err error) {
	// Implement the logic to execute the agent's task
	sys_prompt, err := p.SetupPrompt()
	slog.Info(sys_prompt)
	if err != nil {
		slog.Error(err.Error())
	}
	slog.Info("%s-%s", p.Model, p.Provider)
	provider := providers.GetProvider(p.Provider)
	client := provider.Init()
	system_request := providers.RequestMessage{Role: "system", Content: sys_prompt}
	user_request := providers.RequestMessage{Role: "user", Content: req.Request.Content}

	var steps []*PlannerTask

	provider_req := providers.GenAIProviderRequest{Model: p.Model, Messages: []providers.RequestMessage{system_request, user_request}, Stream: false, ReasoningFormat: "parsed"}
	completion, err := provider.Generate(ctx, client, provider_req)
	if err != nil {
		slog.Error(err.Error())
	}
	err = json.Unmarshal([]byte(completion.Choices[0].Message.Content), &steps)

	if err != nil {
		slog.Error("Failed to unmarshal completion into steps: %v", err.Error())
	}

	payload := &providers.ResponseMessage{Role: "assistant", Content: completion.Choices[0].Message.Content, Thought: completion.Choices[0].Message.Thought}
	response = &YafaiResponse{Source: "planner", Response: payload}

	return response, err
}

func (p *YafaiPlanner) Parse(plan *YafaiResponse) (PlanSteps []*PlannerTask, err error) {

	var steps []*PlannerTask
	planString := plan.Response.Content
	slog.Info("%v", planString)
	slog.Info(planString)
	err = json.Unmarshal([]byte(planString), &steps)

	if err != nil {
		slog.Error("Failed to unmarshal completion into steps: %v", err)
	}

	return steps, err
}

func (p *YafaiPlanner) Close() error {
	// Implement the logic to close the agent's resources
	return nil
}
