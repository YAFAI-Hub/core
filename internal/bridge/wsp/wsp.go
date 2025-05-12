package wsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
	"yafai/internal/nexus/executors"
	"yafai/internal/nexus/providers"
)

func stripJsonDelimiters(rawString string) string {
	startDelimiter := "```json"
	endDelimiter := "```"

	// 1. Trim leading/trailing whitespace from the input string
	trimmed := strings.TrimSpace(rawString)

	// 2. Check if the trimmed string has the specified prefix and suffix
	hasPrefix := strings.HasPrefix(trimmed, startDelimiter)
	hasSuffix := strings.HasSuffix(trimmed, endDelimiter)

	// 3. Ensure the string is long enough to contain more than just the delimiters
	if hasPrefix && hasSuffix && len(trimmed) > len(startDelimiter)+len(endDelimiter) {
		// Extract the content between the delimiters
		content := trimmed[len(startDelimiter) : len(trimmed)-len(endDelimiter)]

		// Trim whitespace from the extracted content itself
		return strings.TrimSpace(content)
	}

	// 4. If delimiters weren't found correctly, return the trimmed original string
	return trimmed
}

func (s *WorkspaceServer) LinkStream(stream WorkspaceService_LinkStreamServer) (err error) { // Assume YourServiceServer and YourService_LinkServer types
	connID := fmt.Sprintf("conn_%d", time.Now().UnixNano())
	slog.Info("New client connected", "connection_id", connID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for Ctrl+C (SIGINT/SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Defer to log closure regardless of how the function exits.
	defer func() {
		go func() {
			<-ctx.Done()
			slog.Info("Context canceled, shutting down")
			slog.Info("Closing stream", "connection_id", connID)
		}()
	}()

	// Outer loop: Receive packets from the client
	for {
		packet, err := stream.Recv()
		if err == io.EOF {
			slog.Info("Client closed the connection", "connection_id", connID)
			return nil
		}
		if err != nil {
			slog.Error("Error receiving packet", "connection_id", connID, "error", err)
			return err
		}

		// Append user message to orchestrator history
		s.Wsp.Orchestrator.AppendChatRecord("user", "orchestrator", packet.Request)
		currentRequest := packet.Request

		// ReACT loop for this packet
		iterationCount := 0

		for {
			// Check for cancellation
			select {
			case <-ctx.Done():
				slog.Error("Stream context cancelled", "connection_id", connID, "error", ctx.Err())
				return ctx.Err()
			default:
			}

			// 1. Plan/Invoke: ask orchestrator what to do
			resp, err := s.InvokeOrchestrator(ctx, &OrchestratorRequest{Request: currentRequest})
			if err != nil {
				slog.Error("Error invoking orchestrator", "connection_id", connID, "error", err)
				stream.Send(&LinkResponse{Response: fmt.Sprintf("Orchestrator Error: %v", err)})
				break
			}

			// 2. Observe: parse orchestrator JSON
			output := stripJsonDelimiters(resp.Response)
			var j map[string]interface{}

			if err := json.Unmarshal([]byte(output), &j); err != nil {
				slog.Error("Error parsing orchestrator response", "connection_id", connID, "error", err)
				stream.Send(&LinkResponse{Response: fmt.Sprintf("Internal Error: %v", err)})
				break
			}

			if msg, ok := j["chat"].(string); ok {
				s.Wsp.Orchestrator.AppendChatRecord("orchestrator", "user", msg)
				stream.Send(&LinkResponse{Response: msg, Trace: "Source: Orchestrator"})
				break
			} else if ans, ok := j["answer"].(string); ok {
				s.Wsp.Orchestrator.AppendChatRecord("orchestrator", "user", ans)
				stream.Send(&LinkResponse{Response: ans, Trace: "Source: Orchestrator"})
				break
			} else if name, ok := j["name"].(string); ok {
				task, _ := j["task"].(string)
				// Append orchestrator plan to history
				s.Wsp.Orchestrator.AppendChatRecord("orchestrator", name, task)

				// Prepare agent request
				agentReq := &executors.YafaiRequest{Request: &providers.RequestMessage{Role: "user", Content: task}}

				// Run agent execution in goroutine and wait
				resultCh := make(chan *executors.YafaiResponse, 1)
				errCh := make(chan error, 1)
				go func() {
					agentExec, exists := s.Wsp.Orchestrator.Team[name]
					if !exists {
						errCh <- fmt.Errorf("agent '%s' not found", name)
						return
					}
					res, err := agentExec.Execute(ctx, agentReq)
					if err != nil {
						errCh <- err
					} else {
						resultCh <- res
					}
				}()

				var agentRes *executors.YafaiResponse
				select {
				case err := <-errCh:
					slog.Error("Agent execution failed", "agent", name, "error", err)
					s.Wsp.Orchestrator.AppendChatRecord(name, "error", err.Error())
					stream.Send(&LinkResponse{Response: fmt.Sprintf("Agent '%s' error: %v", name, err)})
					currentRequest = fmt.Sprintf("Previous agent '%s' failed with error: %s. What's next?", name, err)
					continue

				case agentRes = <-resultCh:
					// Append agent result to history
					content := fmt.Sprintf("Observation: %s (from %s)", agentRes.Response.Content, name)
					s.Wsp.Orchestrator.AppendChatRecord(name, "user", content)
					currentRequest = content
				}
				// Next iteration of the ReACT loop uses updated currentRequest
				iterationCount++

				// If no new response or excessive iterations, terminate the loop
				if iterationCount > 3 {
					slog.Warn("Excessive iterations or no progress made, terminating the loop.")
					stream.Send(&LinkResponse{Response: "Task could not be completed due to repeated failures or no progress."})
					break
				}

				//lastResponse = currentRequest
				continue
			} else {
				slog.Warn("Unexpected orchestrator response format", "response", output)
				stream.Send(&LinkResponse{Response: "Internal Error: Unexpected response format from orchestrator."})
				break
			}

		}
		// Inner loop ends; wait for next packet
	}

	// End of outer receive packet loop
}

func (s *WorkspaceServer) InvokeOrchestrator(ctx context.Context, req *OrchestratorRequest) (resp *OrchestratorResponse, err error) {
	slog.Info("Orchestrator Request:", req.Request)
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

	// steps, err := s.Planner.Parse(planner_resp)

	// if err != nil {
	// 	slog.Error(err.Error())
	// }

	// var response []*PlannerStep

	// for _, step := range steps {
	// 	response = append(response, &PlannerStep{Task: step.Task, Agent: step.Agent, Thought: step.Thought})
	// }
	s.Wsp.Orchestrator.AppendChatRecord("orchestrator", "user", orch_resp.Response.Content)
	slog.Info("Received orchestrator response",
		"connection_id", "connID",
		"response", orch_resp.Response.Content)
	return &OrchestratorResponse{Response: orch_resp.Response.Content}, nil
}

func (s *WorkspaceServer) InvokePlanner(ctx context.Context, req *PlannerRequest) (res *PlannerResponse, err error) {

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

	var response []*PlannerStep

	for _, step := range steps {
		response = append(response, &PlannerStep{Task: step.Task, Agent: step.Agent, Thought: step.Thought})
	}

	return &PlannerResponse{Steps: response}, err

}

func (s *WorkspaceServer) InvokePlanRefine(ctx context.Context, req *PlannerRefineRequest) (res *PlannerResponse, err error) {

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

	var response []*PlannerStep

	for _, step := range steps {
		response = append(response, &PlannerStep{Task: step.Task, Agent: step.Agent, Thought: step.Thought})
	}

	return &PlannerResponse{Steps: response}, err
}
