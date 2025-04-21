package executors

import (
	//"fmt"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"text/template"
	"time"

	skill "yafai/internal/bridge/skill"
	"yafai/internal/nexus/assets/templates"
	"yafai/internal/nexus/providers"

	"google.golang.org/grpc"
)

func ConvertActionsToLLMTools(actions []*skill.Action) []providers.LLMTool {
	var tools []providers.LLMTool

	for _, action := range actions {
		props := make(map[string]providers.LLMProperty)
		var requiredFields []string

		for _, param := range action.Params {
			props[param.Name] = providers.LLMProperty{
				Type:        param.Type,
				Description: param.Description,
			}
			if param.Required {
				requiredFields = append(requiredFields, param.Name)
			}
		}

		tool := providers.LLMTool{
			Type: "function",
			Function: providers.LLMFunction{
				Name:        action.Name,
				Description: action.Description,
				Parameters: providers.LLMFunctionParameters{
					Type:       "object",
					Properties: props,
					Required:   requiredFields,
				},
			},
		}
		tools = append(tools, tool)
	}

	return tools
}

func (a *YafaiAgent) GetInfo() (name string, description string) {
	// Implement the initialization logic for the agent
	return a.Name, a.Description
}

func (a *YafaiAgent) SetupPrompt() (prompt string, err error) {
	// Implement the logic to set up the initial system prompt for the agent
	var tool_desc string

	for _, tool := range a.Tools {
		var output bytes.Buffer
		tmpl, err := template.New("teamDescription").Parse(templates.ToolDescriptionTemplate)
		if err != nil {
			slog.Error(err.Error())
		}
		var data = ToolDescription{Name: tool.Function.Name, Description: tool.Function.Description}
		err = tmpl.Execute(&output, data)
		if err != nil {
			slog.Error(err.Error())
		}

		tool_desc += output.String()
	}

	//history, err := a.getChatHistory()
	if err != nil {
		slog.Error("Parsing chat history failed %s", err.Error())
	}
	system_tmpl, err := template.New("AgentSystem").Parse(templates.AgentTemplate)
	if err != nil {
		slog.Error(err.Error())
	}

	var inst_data = AgentTemplateStruct{Tools: tool_desc, ChatHistory: "", Scratchpad: ""}
	var system_prompt_string bytes.Buffer

	if err != nil {
		slog.Error(err.Error())
	}

	err = system_tmpl.Execute(&system_prompt_string, inst_data)
	if err != nil {
		slog.Error(err.Error())
	}
	//slog.Info(system_prompt_string.String())
	return system_prompt_string.String(), err
}

func (a *YafaiAgent) getChatHistory() (chats string, err error) {

	var historyBuilder strings.Builder
	for _, record := range a.History {
		historyBuilder.WriteString("From: " + record.From + "\n")
		historyBuilder.WriteString("To: " + record.To + "\n")
		historyBuilder.WriteString("Message: " + record.Message + "\n")
		historyBuilder.WriteString("-----\n")
	}
	chats = historyBuilder.String()
	return chats, nil
}

func (a *YafaiAgent) AppendChatRecord(From string, To string, Message string) error {
	// Implement the logicto append a new chat record to the conversation history
	record := &ChatRecord{From: From, To: To, Message: Message}
	a.History = append(a.History, record)
	return nil
}

func (a *YafaiAgent) Parse() (err error) {
	// Implement the logic to parse the agent's response
	return nil
}

func (a *YafaiAgent) ConvertToolCallToExecutionInput(toolCall providers.ToolCall) (ToolExecutionInput, error) {
	var argsMap map[string]string
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &argsMap)
	if err != nil {
		return ToolExecutionInput{}, fmt.Errorf("failed to parse function arguments: %v", err)
	}

	// Find the corresponding action
	var action *skill.Action
	for _, act := range a.Actions {
		if act.Name == toolCall.Function.Name {
			action = act
			break
		}
	}
	if action == nil {
		return ToolExecutionInput{}, fmt.Errorf("action metadata not found for: %s", toolCall.Function.Name)
	}

	// Init all param maps
	pathParams := make(map[string]string)
	queryParams := make(map[string]string)
	bodyParams := make(map[string]string)

	// Distribute params by "in" field
	for _, param := range action.Params {
		val, exists := argsMap[param.Name]
		if !exists {
			continue
		}
		switch param.In {
		case "path":
			pathParams[param.Name] = val
		case "query":
			queryParams[param.Name] = val
		case "body":
			bodyParams[param.Name] = val
		}
	}

	return ToolExecutionInput{
		Name:        action.Name,
		PathParams:  pathParams,
		QueryParams: queryParams,
		BodyParams:  bodyParams,
	}, nil
}

func (a *YafaiAgent) DiscoverTools() (err error) {

	if a.SkillClient != nil && a.Tools != nil {
		//slog.Info("Fetching tools and actions from memory")
		return nil
	}

	conn, err := grpc.Dial("localhost:5001", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// Create a new client
	client := skill.NewSkillServiceClient(conn)

	// Prepare the request
	req := &skill.GetActionRequest{Task: "list all"}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	res, err := client.GetActions(ctx, req)
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

	tools := ConvertActionsToLLMTools(res.Actions)

	a.Tools = tools
	a.Actions = res.Actions

	return err
}

func (a *YafaiAgent) ExecuteTool(req ToolExecutionInput) (res *skill.ExecuteActionResponse, err error) {
	//slog.Info("Recieved tool execution request: %v", req)
	if len(req.QueryParams) == 0 {
		//slog.Info("No Query params passed for the action")
		req.QueryParams = nil // or make(map[string]string) if preferred
	}
	if len(req.PathParams) == 0 {
		//slog.Info("No Path params passed for the action")
		req.PathParams = nil
	}
	if len(req.BodyParams) == 0 {
		//slog.Info("No Body params passed for the action")
		req.BodyParams = nil
	}

	conn, err := grpc.Dial("localhost:5001", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
		return nil, err
	}
	defer conn.Close()

	// Create a new client
	client := skill.NewSkillServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	response, err := client.ExecuteAction(ctx, &skill.ExecuteActionRequest{Name: req.Name, QueryParams: req.QueryParams, PathParams: req.PathParams, BodyParams: req.BodyParams})
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	return response, err
}

func (a *YafaiAgent) extractFinalAnswer(message string) string {
	const marker = "Final Answer:"
	idx := strings.Index(message, marker)
	if idx == -1 {
		return message // fallback: return full message
	}
	return strings.TrimSpace(message[idx+len(marker):])
}

func mapInternalRoleToLLMRole(from string) string {
	switch from {
	case "user":
		return "user"
	case "system":
		return "system"
	default: // agent, orchestrator, tool, etc.
		return "assistant"
	}
}

func extractAfter(input, key string) string {
	idx := strings.Index(input, key)
	if idx == -1 {
		return ""
	}
	return input[idx+len(key):]
}

func (a *YafaiAgent) Execute(ctx context.Context, req *YafaiRequest) (*YafaiResponse, error) {
	// Discover tools
	if err := a.DiscoverTools(); err != nil {
		slog.Error("Tool discovery failed: %v", err)
		return &YafaiResponse{Response: &providers.ResponseMessage{
			Role:    "assistant",
			Content: fmt.Sprintf("Internal error: could not load tools: %v", err),
		}}, err
	}

	// Initialize history if needed
	if req.Source == "orchestrator" {
		a.History = []*ChatRecord{
			{From: "orchestrator", To: "agent", Message: req.Request.Content},
		}
	} else {
		a.History = append(a.History, &ChatRecord{
			From: "orchestrator", To: "agent", Message: req.Request.Content,
		})
	}

	provider := providers.GetProvider(a.Provider)
	client := provider.Init()

	// Set retry parameters
	const maxRetries = 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Build the system prompt: only relevant instructions for the agent
		sysPrompt := a.BuildSystemPrompt()

		// Include the message history in the conversation (not in system prompt)
		conversationHistory := a.BuildConversationHistory()

		providerReq := []providers.RequestMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: fmt.Sprintf("Here's the context so far: %s", conversationHistory)},
		}

		// Generate response from the model
		resp, err := provider.Generate(ctx, client, providers.GenAIProviderRequest{
			Model:    a.Model,
			Messages: providerReq,
			Stream:   false,
			Tools:    a.Tools,
		})

		// Handle model errors
		if err != nil {
			slog.Error("Model error: %v", err)
			return &YafaiResponse{Response: &providers.ResponseMessage{
				Role:    "assistant",
				Content: fmt.Sprintf("Error with the model: %v", err),
			}}, err
		}

		// Ensure the model response is valid
		if resp == nil || len(resp.Choices) == 0 {
			slog.Error("Empty response from the model")
			break
		}

		msg := resp.Choices[0].Message
		content := strings.TrimSpace(msg.Content)
		a.AppendChatRecord("agent", "user", content)

		// Handle clarification and final answer cases
		if strings.HasPrefix(content, "Query:") {
			query := strings.TrimSpace(extractAfter(content, "Query:"))
			return &YafaiResponse{Response: &providers.ResponseMessage{
				Role:    "assistant",
				Content: query,
			}}, nil
		}

		if strings.HasPrefix(content, "Final Answer:") {
			answer := strings.TrimSpace(extractAfter(content, "Final Answer:"))
			return &YafaiResponse{Response: &providers.ResponseMessage{
				Role:    "assistant",
				Content: answer,
			}}, nil
		}

		slog.Info("Tools added : %+v", a.Tools)
		slog.Info("LLM Response : %+v", msg)

		// Handle tool invocation
		if len(msg.ToolCalls) > 0 {
			call := msg.ToolCalls[0]
			thought := fmt.Sprintf("Thought: I need to use tool: %s with input: %s", call.Function.Name, call.Function.Arguments)
			a.AppendChatRecord("assistant", "log", thought)

			action := fmt.Sprintf("Action: %s\nInput: %s", call.Function.Name, call.Function.Arguments)
			a.AppendChatRecord("assistant", "tool", action)

			// Prepare and validate input for the tool
			input, err := a.ConvertToolCallToExecutionInput(call)
			if err != nil {
				return &YafaiResponse{Response: &providers.ResponseMessage{
					Role:    "assistant",
					Content: fmt.Sprintf("Error converting tool input: %v", err),
				}}, err
			}

			// Execute the tool and capture the observation
			result, err := a.ExecuteTool(input)
			if err != nil {
				slog.Error("Tool execution failed: %v", err)
				return &YafaiResponse{Response: &providers.ResponseMessage{
					Role:    "assistant",
					Content: fmt.Sprintf("Error executing tool: %v", err),
				}}, err
			}

			observation := strings.TrimSpace(result.Response)
			a.AppendChatRecord("tool", "assistant", observation)

			// Return the tool observation
			return &YafaiResponse{Response: &providers.ResponseMessage{
				Role:    "tool",
				Content: observation,
			}}, nil
		}

		// If no tool was invoked, ask for clarification or stop
		if attempt == maxRetries {
			return &YafaiResponse{Response: &providers.ResponseMessage{
				Role:    "assistant",
				Content: "I’ve tried several times but still need more details. Could you clarify or provide additional information?",
			}}, nil
		}
	}

	// Fallback message if all retries failed
	return &YafaiResponse{Response: &providers.ResponseMessage{
		Role:    "assistant",
		Content: "I’m unable to complete the task. Please provide more details.",
	}}, nil
}

// BuildSystemPrompt constructs the system prompt with agent instructions (no history)
func (a *YafaiAgent) BuildSystemPrompt() string {
	// Static instructions for the system prompt (no history here)
	return "You are a helpful assistant. Please assist the user in completing the requested task. Ask for clarification when needed."
}

// BuildConversationHistory concatenates chat history for use as context
func (a *YafaiAgent) BuildConversationHistory() string {
	var historyBuilder strings.Builder
	for _, record := range a.History {
		if record.From == "orchestrator" {
			historyBuilder.WriteString(fmt.Sprintf("User: %s\n", record.Message))
		} else if record.From == "agent" {
			historyBuilder.WriteString(fmt.Sprintf("Assistant: %s\n", record.Message))
		} else if record.From == "tool" {
			historyBuilder.WriteString(fmt.Sprintf("Tool: %s\n", record.Message))
		}
	}
	return historyBuilder.String()
}

// Ensure AppendChatRecord and extractFinalAnswer helper functions are defined elsewhere
// func (a *YafaiAgent) AppendChatRecord(from, to, message string) { ... }
// func (a *YafaiAgent) extractFinalAnswer(content string) string { ... }
