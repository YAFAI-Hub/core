package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"

	wsp "yafai/internal/bridge/wsp"
)

type webSocketHandler struct {
	workspaceClient wsp.WorkspaceServiceClient
	upgrader        websocket.Upgrader
}

func NewChatWebSocketHandler(
	workspaceClient wsp.WorkspaceServiceClient,

) *webSocketHandler {
	return &webSocketHandler{
		workspaceClient: workspaceClient,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement strict origin checking
				return true
			},
		},
	}
}

func (h *webSocketHandler) HandleChatWebSocket(
	w http.ResponseWriter,
	r *http.Request,
	userID string,
) {
	// Upgrade WebSocket connection
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade websocket", "error", err)
		return
	}
	defer conn.Close()

	// Generate connection ID
	connID := fmt.Sprintf("chat_ws_conn_%d", time.Now().UnixNano())
	slog.Info("New WebSocket client connected",
		"connection_id", connID,
		"user_id", userID,
	)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for connection closure or interruption
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Shutdown signal received")
		cancel() // triggers context cancellation
	}()

	// Establish gRPC stream
	stream, err := h.workspaceClient.LinkStream(ctx)
	if err != nil {
		slog.Error("Failed to create workspace stream", "error", err)
		return
	}
	defer stream.CloseSend()

	// Error channel to capture goroutine errors
	errChan := make(chan error, 2)

	// Done channel to signal stream goroutine completion
	doneChan := make(chan struct{})

	// Goroutine to receive from workspace and forward to WebSocket client
	go func() {
		defer close(doneChan)
		for {
			resp, err := stream.Recv()
			if err != nil {
				errChan <- fmt.Errorf("workspace recv failed: %w", err)
				return
			}

			slog.Info("Got response from workspace", "response", resp.Response)

			// Forward response to WebSocket client
			if err := conn.WriteMessage(websocket.TextMessage, []byte(resp.Response)); err != nil {
				errChan <- fmt.Errorf("websocket send failed: %w", err)
				return
			}
		}
	}()

	// Main loop: receive from client, send to workspace
	for {
		select {
		case <-ctx.Done():
			slog.Info("Context canceled, shutting down websocket")
			return
		case err := <-errChan:
			slog.Error("Error in websocket communication", "error", err)
			return
		case <-doneChan:
			slog.Info("Workspace receiver goroutine exited unexpectedly")
			return
		default:
			// Read message from WebSocket client
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					slog.Error("Websocket read error", "error", err)
				}
				return
			}

			// Extract thread from query parameter
			thread := r.URL.Query().Get("thread")
			if thread == "" {
				thread = "default"
			}

			slog.Info("Forwarding message to workspace",
				"message", string(message),
				"thread", thread,
			)

			// Send message to workspace
			err = stream.Send(&wsp.LinkRequest{
				Request: string(message),
			})

			if err != nil {
				slog.Error("Error sending message to workspace", "error", err)
				return
			}
		}
	}
}

// Singleton instance
var WebSocketHandler *webSocketHandler
