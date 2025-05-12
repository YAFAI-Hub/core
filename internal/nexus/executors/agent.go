package executors

import (
	//"fmt"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"text/template"
	"time"

	skill "yafai/internal/bridge/skill"
	"yafai/internal/nexus/assets/templates"
	"yafai/internal/nexus/providers"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

func buildProperties(params []*skill.Parameter) (map[string]providers.LLMProperty, []string) {
	properties := make(map[string]providers.LLMProperty)
	var required []string

	for _, param := range params {
		prop := providers.LLMProperty{
			Type:        param.Type,
			Description: param.Description,
		}

		// If the parameter has enum values, include them in the property
		if len(param.Enum) > 0 {
			prop.Enum = param.Enum
		}

		// Handle properties (nested parameters for objects and arrays)
		if len(param.Properties) > 0 || len(param.Items) > 0 {
			var subProps map[string]providers.LLMProperty
			var subRequired []string

			// If the parameter is an array, handle its Items (nested properties of the array elements)
			if param.Type == "array" {
				subProps, subRequired = buildProperties(param.Items)
				prop.Items = &providers.LLMProperty{
					Type:       "object", // Arrays contain objects (or can be expanded)
					Properties: subProps,
					Required:   subRequired,
				}
			} else if param.Type == "object" {
				// If the parameter is an object, handle its Properties recursively
				subProps, subRequired = buildProperties(param.Properties)
				prop.Properties = subProps
				prop.Required = subRequired
			}
		}

		// Store the final property in the map
		properties[param.Name] = prop

		// Collect required fields
		if param.Required {
			required = append(required, param.Name)
		}
	}

	return properties, required
}

func ConvertActionsToLLMTools(actions []*skill.Action) []providers.LLMTool {
	var tools []providers.LLMTool

	for _, action := range actions {
		// Convert properties and collect required fields using the external function
		props, requiredFields := buildProperties(action.Params)

		// Create the tool from the action
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
			Strict: true,
		}

		// Append the tool to the list
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
	var argsMap map[string]interface{}
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &argsMap)
	if err != nil {
		return ToolExecutionInput{}, fmt.Errorf("failed to parse function arguments: %v", err)
	}

	// Find matching action
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

	// Initialize param buckets
	pathParams := make(map[string]interface{})
	queryParams := make(map[string]interface{})
	bodyParams := make(map[string]interface{})

	// Recursive function to collect nested values
	var extractValue func(param *skill.Parameter, value interface{}) interface{}
	extractValue = func(param *skill.Parameter, value interface{}) interface{} {
		if value == nil || len(param.Properties) == 0 {
			return value
		}

		switch param.Type {
		case "object":
			// Recurse into object properties
			valMap, ok := value.(map[string]interface{})
			if !ok {
				return value
			}
			result := make(map[string]interface{})
			for _, subParam := range param.Properties {
				if subVal, exists := valMap[subParam.Name]; exists {
					result[subParam.Name] = extractValue(subParam, subVal)
				}
			}
			return result

		case "array":
			// Recurse into each element
			valSlice, ok := value.([]interface{})
			if !ok {
				return value
			}
			var result []interface{}
			for _, item := range valSlice {
				result = append(result, extractValue(param.Properties[0], item))
			}
			return result

		default:
			return value
		}
	}

	// Distribute parameters based on "in" field
	for _, param := range action.Params {
		val, exists := argsMap[param.Name]
		if !exists {
			continue
		}
		extracted := extractValue(param, val)

		switch param.In {
		case "path":
			pathParams[param.Name] = extracted
		case "query":
			queryParams[param.Name] = extracted
		case "body":
			bodyParams[param.Name] = extracted
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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	skill_root := fmt.Sprintf("%s/.yafai/plugins/skill.sock", homeDir)
	slog.Info(skill_root)
	conn, err := grpc.Dial(fmt.Sprintf("unix:%s", skill_root), grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	slog.Info("----------------------------------------------")
	slog.Info("Tools discovered: ", a.Tools)
	slog.Info("----------------------------------------------")
	return err
}

func toStructPB(value interface{}) (*structpb.Value, error) {
	switch v := value.(type) {
	case string:
		return structpb.NewStringValue(v), nil
	case bool:
		return structpb.NewBoolValue(v), nil
	case float64:
		return structpb.NewNumberValue(v), nil
	case []interface{}:
		// Handle arrays (convert each element to structpb.Value)
		list := make([]*structpb.Value, len(v))
		for i, item := range v {
			pbVal, err := toStructPB(item) // Recursively handle each element
			if err != nil {
				return nil, err
			}
			list[i] = pbVal
		}
		return structpb.NewListValue(&structpb.ListValue{Values: list}), nil
	case map[string]interface{}:
		// Handle maps (convert each key-value pair to structpb.Value)
		fields := map[string]*structpb.Value{}
		for key, item := range v {
			pbVal, err := toStructPB(item) // Recursively handle nested maps
			if err != nil {
				return nil, err
			}
			fields[key] = pbVal
		}
		return structpb.NewStructValue(&structpb.Struct{Fields: fields}), nil
	default:
		// Default case for unknown types (convert to string)
		return structpb.NewStringValue(fmt.Sprintf("%v", v)), nil
	}
}

func (a *YafaiAgent) ExecuteTool(req ToolExecutionInput) (*skill.ExecuteActionResponse, error) {
	// Helper: Convert any interface{} to *structpb.Value for protobuf
	toStructPB := func(value interface{}) (*structpb.Value, error) {
		switch v := value.(type) {
		case string:
			return structpb.NewStringValue(v), nil
		case bool:
			return structpb.NewBoolValue(v), nil
		case float64:
			return structpb.NewNumberValue(v), nil
		case []interface{}:
			// Handle arrays (convert each element to structpb.Value)
			list := make([]*structpb.Value, len(v))
			for i, item := range v {
				pbVal, err := toStructPB(item) // Recursively handle each element
				if err != nil {
					return nil, err
				}
				list[i] = pbVal
			}
			return structpb.NewListValue(&structpb.ListValue{Values: list}), nil
		case map[string]interface{}:
			// Handle maps (convert each key-value pair to structpb.Value)
			fields := make(map[string]*structpb.Value)
			for key, item := range v {
				pbVal, err := toStructPB(item) // Recursively handle nested maps
				if err != nil {
					return nil, err
				}
				fields[key] = pbVal
			}
			return structpb.NewStructValue(&structpb.Struct{Fields: fields}), nil
		default:
			// Default case for unknown types (convert to string)
			return structpb.NewStringValue(fmt.Sprintf("%v", v)), nil
		}
	}

	// Helper: Convert map[string]interface{} to Struct fields
	convertMap := func(input map[string]interface{}) (map[string]*structpb.Value, error) {
		fields := make(map[string]*structpb.Value)
		for key, val := range input {
			pbVal, err := toStructPB(val)
			if err != nil {
				return nil, fmt.Errorf("failed to convert key '%s': %w", key, err)
			}
			fields[key] = pbVal
		}
		return fields, nil
	}

	// Convert parameters (query, path, and body params)
	queryFields, err := convertMap(req.QueryParams)
	if err != nil {
		return nil, fmt.Errorf("error processing query params: %w", err)
	}
	pathFields, err := convertMap(req.PathParams)
	if err != nil {
		return nil, fmt.Errorf("error processing path params: %w", err)
	}
	bodyFields, err := convertMap(req.BodyParams)
	if err != nil {
		return nil, fmt.Errorf("error processing body params: %w", err)
	}

	// Socket path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	skillSocket := fmt.Sprintf("%s/.yafai/plugins/skill.sock", homeDir)
	slog.Info("Connecting to socket:", skillSocket)

	// gRPC connection
	conn, err := grpc.Dial(fmt.Sprintf("unix:%s", skillSocket), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC socket: %w", err)
	}
	defer conn.Close()

	client := skill.NewSkillServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build request
	reqStruct := &skill.ExecuteActionRequest{
		Name:        req.Name,
		QueryParams: &structpb.Struct{Fields: queryFields},
		PathParams:  &structpb.Struct{Fields: pathFields},
		BodyParams:  &structpb.Struct{Fields: bodyFields},
	}

	// Execute action
	response, err := client.ExecuteAction(ctx, reqStruct)
	if err != nil {
		slog.Error("ExecuteAction failed:", err)
		return nil, err
	}

	return response, nil
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
		sysPrompt, err := a.SetupPrompt()
		if err != nil {
			slog.Error("Failed to set up system prompt: %v", err)
			return &YafaiResponse{Response: &providers.ResponseMessage{
				Role:    "assistant",
				Content: fmt.Sprintf("Error setting up system prompt: %v", err),
			}}, err
		}

		// Include the message history in the conversation (not in the system prompt)
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

		// Extract message content
		msg := resp.Choices[0].Message
		content := strings.TrimSpace(msg.Content)
		a.AppendChatRecord("agent", "user", content)

		// Handle clarification and final answer cases
		if strings.Contains(content, "Query:") {
			query := strings.TrimSpace(extractAfter(content, "Query:"))
			return &YafaiResponse{Response: &providers.ResponseMessage{
				Role:    "assistant",
				Content: query,
			}}, nil
		}

		if strings.Contains(content, "Final Answer:") {
			answer := strings.TrimSpace(extractAfter(content, "Final Answer:"))
			return &YafaiResponse{Response: &providers.ResponseMessage{
				Role:    "assistant",
				Content: answer,
			}}, nil
		}

		// Log the response
		slog.Info("LLM Response: %+v", msg)

		// Handle tool invocation
		if len(msg.ToolCalls) > 0 {
			call := msg.ToolCalls[0]
			thought := fmt.Sprintf("Thought: I need to use tool: %s with input: %s", call.Function.Name, call.Function.Arguments)
			slog.Info("Tool call: %s", thought)
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
