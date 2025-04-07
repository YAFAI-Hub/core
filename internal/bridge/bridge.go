package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"time"
	link "yafai/internal/bridge/proto"
	"yafai/internal/nexus/executors"
	"yafai/internal/nexus/providers"
)

func (s *LinkServer) ChatStream(stream link.ChatService_ChatStreamServer) (err error) {
	// Generate a unique connection ID for logging
	connID := fmt.Sprintf("conn_%d", time.Now().UnixNano())
	slog.Info("New client connected", "connection_id", connID)

	defer func() {
		slog.Info("Closing stream", "connection_id", connID)
		// Don't send closing message if the connection is already closed
		if err == nil {
			if sendErr := stream.Send(&link.ChatResponse{Response: "Connection closed by server"}); sendErr != nil {
				slog.Info("Could not send closing message",
					"connection_id", connID,
					"error", sendErr)
			}
		}
	}()

	for {
		packet, err := stream.Recv()
		if err == io.EOF {
			slog.Info("Client closed the connection", "connection_id", connID)
			return nil // Return cleanly on client disconnect
		}
		if err != nil {
			slog.Error("Error receiving packet",
				"connection_id", connID,
				"error", err)
			return err // Return on other errors to close the stream
		}

		record := &executors.ChatRecord{From: "user", To: "orchestrator", Message: packet.Request}
		s.Wsp.Orchestrator.AppendChatRecord(record.From, record.To, record.Message)
		slog.Info("Received packet",
			"connection_id", connID,
			"packet", packet.Request)

		ctx := context.Background()

		//channel to handle errors from goroutine
		errChan := make(chan error, 1)

		go func(packet *link.ChatRequest) {
			resp, err := s.InvokeOrchestrator(ctx, &link.OrchestratorRequest{Request: packet.Request})
			if err != nil {
				slog.Error("Error invoking orchestrator",
					"connection_id", connID,
					"error", err.Error())
				errChan <- err
				return
			}

			jsonResp := []byte(resp.Response)
			var jsonMap map[string]interface{}

			if err := json.Unmarshal(jsonResp, &jsonMap); err != nil {
				slog.Error("Error unmarshaling JSON response",
					"connection_id", connID,
					"error", err)
				errChan <- err
				return
			}

			if chatMessage, ok := jsonMap["chat"].(string); ok {
				slog.Info("chat message received", "connection_id", connID, "message", chatMessage)
				if err := stream.Send(&link.ChatResponse{
					Response: chatMessage,
					Trace:    "Source : YAFAI-Orchestrator",
				}); err != nil {
					slog.Error("Error sending chat response",
						"connection_id", connID,
						"error", err)
					errChan <- err
					return
				}
			}

			if invokePlanner, ok := jsonMap["invoke_planner"].(bool); ok && invokePlanner {
				slog.Info("invoke_planner is true",
					"connection_id", connID)
				req := fmt.Sprintf("Here is some context : %v. Now %s", s.Wsp.Orchestrator.History, packet.Request)
				plan, err := s.InvokePlanner(ctx, &link.PlannerRequest{Request: req})
				if err != nil {
					slog.Error("Error invoking planner",
						"connection_id", connID,
						"error", err.Error())
					errChan <- err
					return
				}
				s.Wsp.Orchestrator.AppendChatRecord("planner", "user", "Plan submitted for user review")

				for _, step := range plan.Steps {
					slog.Info("Executing planner step",
						"connection_id", connID,
						"task", step.Task,
						"agent", step.Agent,
						"thought", step.Thought)

					// Append the planner step to the chat record

					// Send the planner step as a response to the client
					if err := stream.Send(&link.ChatResponse{
						Response: fmt.Sprintf("Planner Step - Task: %s, Agent: %s, Thought: %s", step.Task, step.Agent, step.Thought),
						Trace:    "Source : YAFAI-Planner",
					}); err != nil {
						slog.Error("Error sending planner step response",
							"connection_id", connID,
							"error", err)
						errChan <- err
						return
					}
				}
				// Add logic to handle invoke_planner being true
			} else {
				slog.Info("invoke_planner is false or missing",
					"connection_id", connID)
			}
			if executePlan, ok := jsonMap["execute_plan"].(bool); ok && executePlan {
				slog.Info("execute_plan is true", "connection_id", connID)

				plan := s.Wsp.Orchestrator.Plan
				if plan == nil {
					slog.Error("No plan available to execute", "connection_id", connID)
					errChan <- fmt.Errorf("no plan available to execute")
					return
				}
				agent_log := make(map[string]string)
				for _, step := range plan.Response {
					slog.Info("Executing plan step",
						"connection_id", connID,
						"task", step.Task,
						"agent", step.Agent,
						"thought", step.Thought,
						"depends_on", step.DependsOn)

					// Execute the step
					if step.DependsOn == "" {
						slog.Info("Step has no dependency",
							"connection_id", connID,
							"depends_on", step.DependsOn)
					} else {
						slog.Info("Step has dependency",
							"connection_id", connID,
							"depends_on", step.DependsOn)
						if result, ok := agent_log[step.DependsOn]; ok {
							slog.Info("Dependency result found",
								"connection_id", connID,
								"result", result)
							step.Task = fmt.Sprintf("The output from  %s is \n %s \n. Now, %s ", step.Agent, result, step.Task)
						} else {
							slog.Error("Dependency result not found",
								"connection_id", connID,
								"dependency", step.DependsOn)
							errChan <- fmt.Errorf("dependency result not found for %s", step.DependsOn)
							return
						}

					}
					result, err := s.Wsp.Orchestrator.Team[step.Agent].Execute(ctx, &executors.YafaiRequest{Request: &providers.RequestMessage{Role: "user", Content: step.Task}})
					agent_log[step.Agent] = result.Response.Content
					if err != nil {
						slog.Error("Error executing plan step",
							"connection_id", connID,
							"task", step.Task,
							"error", err)
						errChan <- err
						return
					}

					// Append the execution result to the chat record

					s.Wsp.Orchestrator.AppendChatRecord(step.Agent, "user", result.Response.Content)

					// Send the execution result as a response to the client
					if err := stream.Send(&link.ChatResponse{
						Response: result.Response.Content,
						Trace:    fmt.Sprintf("Source : %s-Agent", step.Agent),
					}); err != nil {
						slog.Error("Error sending execution result response",
							"connection_id", connID,
							"error", err)
						errChan <- err
						return
					}
				}
			} else {
				slog.Info("execute_plan is false or missing", "connection_id", connID)
			}

			if refinePlan, ok := jsonMap["refine_plan"].(bool); ok && refinePlan {
				slog.Info("refine_plan is true", "connection_id", connID)

				plan := s.Wsp.Orchestrator.Plan
				if plan == nil {
					slog.Error("No plan available to refine", "connection_id", connID)
					errChan <- fmt.Errorf("no plan available to refine")
					return
				}
				var plan_string string
				for _, step := range plan.Response {
					plan_string += fmt.Sprintf("Task: %s, Agent: %s, Thought: %s\n", step.Task, step.Agent, step.Thought)
				}
				refine_req := &link.PlannerRefineRequest{Plan: plan_string, Refinement: packet.Request}
				refinedPlan, err := s.InvokePlanRefine(ctx, refine_req)
				if err != nil {
					slog.Error("Error refining plan",
						"connection_id", connID,
						"error", err)
					errChan <- err
					return
				}

				s.Wsp.Orchestrator.AppendChatRecord("orchestrator", "user", "Plan refined successfully")

				for _, step := range refinedPlan.Steps {
					slog.Info("Refined plan step",
						"connection_id", connID,
						"task", step.Task,
						"agent", step.Agent,
						"thought", step.Thought)

					// Send the refined plan step as a response to the client
					if err := stream.Send(&link.ChatResponse{
						Response: fmt.Sprintf("Refined Plan Step - Task: %s, Agent: %s, Thought: %s", step.Task, step.Agent, step.Thought),
						Trace:    "Source : YAFAI-Planer Refine",
					}); err != nil {
						slog.Error("Error sending refined plan step response",
							"connection_id", connID,
							"error", err)
						errChan <- err
						return
					}
				}
			} else {
				slog.Info("refine_plan is false or missing", "connection_id", connID)
			}

			errChan <- nil
		}(packet)

		// Wait for goroutine to complete or context to be cancelled
		select {
		case err := <-errChan:
			if err != nil {
				slog.Error("Error in stream processing",
					"connection_id", connID,
					"error", err)
				// Continue serving other requests even if this one failed
				continue
			}
		case <-ctx.Done():
			slog.Error("Context cancelled",
				"connection_id", connID,
				"error", ctx.Err())
			return ctx.Err()
		}
	}
}

func (s *LinkServer) InvokeOrchestrator(ctx context.Context, req *link.OrchestratorRequest) (resp *link.OrchestratorResponse, err error) {
	orch_resp, err := s.Wsp.Orchestrator.Execute(ctx, &executors.YafaiRequest{Request: &providers.RequestMessage{Role: "user", Content: req.Request}})

	// re := regexp.MustCompile(`<think>(.*?)</think>`)
	// output := re.ReplaceAllString(planner_resp.Response.Content, "")

	// // Remove any leading/trailing whitespace
	// output = strings.TrimSpace(output)

	// // Find the beginning of the json array
	// start := strings.Index(output, "[")

	// if start == -1 {
	// 	slog.Info("No JSON array found.")
	// 	return
	// }

	// //Extract the JSON array string
	// planner_resp.Response.Content = output[start:]
	if err != nil {
		slog.Error(err.Error())
	}

	// steps, err := s.Wsp.Planner.Parse(planner_resp)

	// if err != nil {
	// 	slog.Error(err.Error())
	// }

	// var response []*link.PlannerStep

	// for _, step := range steps {
	// 	response = append(response, &link.PlannerStep{Task: step.Task, Agent: step.Agent, Thought: step.Thought})
	// }
	s.Wsp.Orchestrator.AppendChatRecord("orchestrator", "user", orch_resp.Response.Content)
	slog.Info("Received orchestrator response",
		"connection_id", "connID",
		"response", orch_resp.Response.Content)
	return &link.OrchestratorResponse{Response: orch_resp.Response.Content}, nil
}

func (s *LinkServer) InvokePlanner(ctx context.Context, req *link.PlannerRequest) (res *link.PlannerResponse, err error) {

	planner_resp, err := s.Wsp.Planner.Execute(ctx, &executors.YafaiRequest{Request: &providers.RequestMessage{Role: "user", Content: req.Request}})

	re := regexp.MustCompile(`<think>(.*?)</think>`)
	output := re.ReplaceAllString(planner_resp.Response.Content, "")

	// Remove any leading/trailing whitespace
	output = strings.TrimSpace(output)

	// Find the beginning of the json array
	start := strings.Index(output, "[")

	if start == -1 {
		slog.Info("No JSON array found.")
		return
	}

	//Extract the JSON array string
	planner_resp.Response.Content = output[start:]
	if err != nil {
		slog.Error(err.Error())
	}

	steps, err := s.Wsp.Planner.Parse(planner_resp)
	s.Wsp.Orchestrator.UpdatePlan(&executors.PlannerResponse{Response: steps})
	if err != nil {
		slog.Error(err.Error())
	}

	var response []*link.PlannerStep

	for _, step := range steps {
		response = append(response, &link.PlannerStep{Task: step.Task, Agent: step.Agent, Thought: step.Thought})
	}

	return &link.PlannerResponse{Steps: response}, err

}

func (s *LinkServer) InvokePlanRefine(ctx context.Context, req *link.PlannerRefineRequest) (res *link.PlannerResponse, err error) {

	refinement_payload := fmt.Sprintf("Refine the following plan,\n %s \n based on the refinement request: %s. Stick to the formatting isntructions", req.Plan, req.Refinement)
	planner_resp, err := s.Wsp.Planner.Execute(ctx, &executors.YafaiRequest{Request: &providers.RequestMessage{Role: "user", Content: refinement_payload}})

	re := regexp.MustCompile(`<think>(.*?)</think>`)
	output := re.ReplaceAllString(planner_resp.Response.Content, "")

	// Remove any leading/trailing whitespace
	output = strings.TrimSpace(output)

	// Find the beginning of the json array
	start := strings.Index(output, "[")

	if start == -1 {
		slog.Info("No JSON array found.")
		return
	}

	//Extract the JSON array string
	planner_resp.Response.Content = output[start:]
	if err != nil {
		slog.Error(err.Error())
	}

	steps, err := s.Wsp.Planner.Parse(planner_resp)
	s.Wsp.Orchestrator.UpdatePlan(&executors.PlannerResponse{Response: steps})
	if err != nil {
		slog.Error(err.Error())
	}

	var response []*link.PlannerStep

	for _, step := range steps {
		response = append(response, &link.PlannerStep{Task: step.Task, Agent: step.Agent, Thought: step.Thought})
	}

	return &link.PlannerResponse{Steps: response}, err
}
