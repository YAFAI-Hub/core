package wsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	db "yafai/internal/bridge/db"
	"yafai/internal/nexus/executors"
	"yafai/internal/nexus/providers"
	"yafai/internal/nexus/workspace"
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

func getCurrentWorkspace(wsp *db.Workspace) (wspCore *workspace.Workspace) {
	CoreTeam := make(map[string]*executors.YafaiAgent)
	slog.Info("+%v", wsp)
	for _, agent := range wsp.Orchestrator.Team {
		provider := providers.GetProvider(agent.Provider)
		c_agent := &executors.YafaiAgent{
			Name:          agent.Name,
			Description:   agent.Description,
			Capabilities:  agent.Capabilities,
			Model:         agent.Model,
			Provider:      agent.Provider,
			GenAIProvider: provider,
			Goal:          agent.Goal,
		}
		CoreTeam[agent.Name] = c_agent
	}

	orch_provider := providers.GetProvider(wsp.Orchestrator.Provider)
	CoreOrchestrator := &executors.YafaiOrchestrator{
		Name:          wsp.Orchestrator.Name,
		Description:   wsp.Orchestrator.Description,
		Scope:         wsp.Orchestrator.Scope,
		Model:         wsp.Orchestrator.Model,
		Goal:          wsp.Orchestrator.Goal,
		Provider:      wsp.Orchestrator.Provider,
		GenAIProvider: orch_provider,
		Team:          CoreTeam,
	}

	CoreWsp := &workspace.Workspace{
		Name:         wsp.Name,
		Orchestrator: CoreOrchestrator,
	}

	return CoreWsp
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

		// Convert WspId and ThreadId from string to int64
		wspID, err := strconv.ParseInt(packet.WspId, 10, 64)
		if err != nil {
			slog.Error("Invalid WspId", "value", packet.WspId, "error", err)
			return err
		}
		threadID, err := strconv.ParseInt(packet.ThreadId, 10, 64)
		if err != nil {
			slog.Error("Invalid ThreadId", "value", packet.ThreadId, "error", err)
			return err
		}

		wsp_db, err := s.Db.GetWorkspaceByID(ctx, wspID)
		if err != nil {
			slog.Error("Error in getting the current workspace", err.Error())
		}
		CoreWsp := getCurrentWorkspace(wsp_db)
		// Append user message to orchestrator history

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
			resp, err := InvokeOrchestrator(ctx, CoreWsp, s.Db, threadID, &OrchestratorRequest{Request: packet.Request})

			if err != nil {
				slog.Error(err.Error())
			}

			slog.Info("Orchestrator Response:", resp)

			if err != nil {
				slog.Error("Error invoking orchestrator", "connection_id", connID, "error", err)
				stream.Send(&LinkResponse{Response: fmt.Sprintf("Orchestrator Error: %v", err)})
				break
			}

			// 2. Observe: parse orchestrator JSON
			//output := stripJsonDelimiters(resp.)
			//var j map[string]interface{}

			if err != nil {
				slog.Error("Error parsing orchestrator response", "connection_id", connID, "error", err)
				stream.Send(&LinkResponse{Response: fmt.Sprintf("Internal Error: %v", err)})
				break
			}

			if resp.Chat != "" {
				message := db.Message{From: "orchestrator", To: "user", Content: resp.Chat}
				CoreWsp.Orchestrator.AppendChatRecord(s.Db, threadID, message)
				stream.Send(&LinkResponse{Response: resp.Chat, Trace: "Source: Orchestrator"})
				break
			} else if resp.Answer != "" {
				//CoreWsp.Orchestrator.AppendChatRecord("orchestrator", "user", resp.Answer)
				stream.Send(&LinkResponse{Response: resp.Answer, Trace: "Source: Orchestrator"})
				break
			} else if resp.Step != nil {
				// Append orchestrator plan to history
				message := db.Message{From: "orchestrator", To: "user", Content: resp.Chat}
				CoreWsp.Orchestrator.AppendChatRecord(s.Db, threadID, message)

				// Prepare agent request
				agentReq := &executors.YafaiRequest{Request: &providers.RequestMessage{Role: "user", Content: resp.Step.Task}}

				// Run agent execution in goroutine and wait
				resultCh := make(chan *executors.YafaiResponse, 1)
				errCh := make(chan error, 1)
				go func() {
					agentExec, exists := CoreWsp.Orchestrator.Team[resp.Step.Name]
					if !exists {
						errCh <- fmt.Errorf("agent '%s' not found", resp.Step.Name)
						return
					}
					res, err := agentExec.Execute(ctx, agentReq)
					if err != nil {
						errCh <- err
					} else {
						resultCh <- res
					}
				}()
				//var currentRequest string;
				var agentRes *executors.YafaiResponse
				select {

				case err := <-errCh:
					slog.Error("Agent execution failed", "agent", resp.Step.Name, "error", err)
					message := db.Message{From: resp.Step.Name, To: "user", Content: err.Error()}
					CoreWsp.Orchestrator.AppendChatRecord(s.Db, threadID, message)
					stream.Send(&LinkResponse{Response: fmt.Sprintf("Agent '%s' error: %v", resp.Step.Name, err)})
					continue

				case agentRes = <-resultCh:
					// Append agent result to history
					content := fmt.Sprintf("Observation: %s (from %s)", agentRes.Response.Content, resp.Step.Name)
					message := db.Message{From: resp.Step.Name, To: "user", Content: content}
					CoreWsp.Orchestrator.AppendChatRecord(s.Db, threadID, message)
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
				slog.Warn("Unexpected orchestrator response format", "response", resp)
				stream.Send(&LinkResponse{Response: "Internal Error: Unexpected response format from orchestrator."})
				break
			}

		}
		// Inner loop ends; wait for next packet
	}

	// End of outer receive packet loop
}

func InvokeOrchestrator(ctx context.Context, CoreWsp *workspace.Workspace, db_conn *db.DBWrapper, thread int64, req *OrchestratorRequest) (resp *executors.YafaiOrchestratorResponse, err error) {

	orch_resp, err := CoreWsp.Orchestrator.Execute(ctx, db_conn, thread, &executors.YafaiRequest{Request: &providers.RequestMessage{Role: "user", Content: req.Request}})
	if err != nil {
		slog.Error("Orchestrator Invokation Error: %s", err.Error(), "error")
	}
	slog.Info("Orchestrator Response:%s", orch_resp.Response.Content, "orchestrator")

	var response executors.YafaiOrchestratorResponse

	if err := json.Unmarshal([]byte(orch_resp.Response.Content), &response); err != nil {
		slog.Error("Error decoding orchestrator response", "error", err)
		return nil, fmt.Errorf("failed to decode orchestrator response: %w", err)
	}

	slog.Info("Received orchestrator response", "response", response)

	return &response, nil
}
