package link

import (
	context "context"
	"fmt"
	"io"
	//"log"
	"log/slog"
	//"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	//"yafai/internal/bridge/relay"
	wsp "yafai/internal/bridge/wsp"

	//"github.com/gorilla/websocket"
	grpc "google.golang.org/grpc"
)

func NewGrpcLink(workspaceServerAddress string) (*LinkServer, error) {
	// Establish a single gRPC connection to the workspace server that will be shared by all clients.
	// In production, use secure connections (e.g., grpc.WithTransportCredentials).
	conn, err := grpc.Dial(workspaceServerAddress, grpc.WithInsecure()) // Use grpc.WithTransportCredentials in production
	if err != nil {
		return nil, fmt.Errorf("failed to dial workspace server: %w", err)
	}
	// In a real app, you'd want to defer conn.Close() when the server shuts down

	// 2. Create a workspace service client
	// 3. Call the LinkStream RPC to get the stream instance
	// You might need a context here, depending on the actual RPC signature

	// 4. Initialize the LinkServer struct with the obtained stream
	server := &LinkServer{
		WspConn: conn,
		// UnimplementedChatServiceServer is usually embedded, no explicit value needed
	}

	return server, nil

}

// NewHttpLink fallback for webclients
// func NewHttpLink(workspaceServerAddress string) http.HandlerFunc {

// 	relayRouter, err := relay.SetupRelay(relay.Config{
// 		MongoURI:     "mongodb://localhost:27017",
// 		DatabaseName: "yafai",
// 	})
// 	if err != nil {
// 		log.Fatalf("Failed to setup relay: %v", err)
// 	}

// 	return func(w http.ResponseWriter, r *http.Request) {
// 		if r.Header.Get("Upgrade") == "websocket" {
// 			// Handle WebSocket connection

// 			upgrader := websocket.Upgrader{
// 				ReadBufferSize:  1024,
// 				WriteBufferSize: 1024,
// 				CheckOrigin: func(r *http.Request) bool {
// 					return true // Allow all origins.  In production, you'd want to check the origin.
// 				},
// 			}

// 			conn, err := upgrader.Upgrade(w, r, nil)
// 			if err != nil {
// 				slog.Error("Failed to upgrade websocket", "error", err)
// 				return
// 			}
// 			defer conn.Close()
// 			slog.Info("Websocket connection established", "remote_addr", conn.RemoteAddr())
// 			// Generate a unique connection ID for logging
// 			connID := fmt.Sprintf("ws_conn_%d", time.Now().UnixNano())
// 			slog.Info("New websocket client connected", "connection_id", connID)

// 			// Establish gRPC connection to the workspace server
// 			grpcConn, err := grpc.Dial(workspaceServerAddress, grpc.WithInsecure())
// 			if err != nil {
// 				slog.Error("Failed to dial workspace server", "error", err)
// 				return
// 			}
// 			defer grpcConn.Close()

// 			wspClient := wsp.NewWorkspaceServiceClient(grpcConn)

// 			ctx, cancel := context.WithCancel(context.Background())
// 			defer cancel()

// 			// Listen for Ctrl+C (SIGINT/SIGTERM)
// 			sigChan := make(chan os.Signal, 1)
// 			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

// 			go func() {
// 				<-sigChan
// 				slog.Info("Shutdown signal received")
// 				cancel() // triggers context cancellation
// 			}()

// 			wspStream, err := wspClient.LinkStream(ctx)
// 			if err != nil {
// 				slog.Error("Failed to create new stream", "error", err)
// 				return
// 			}

// 			defer func() {
// 				slog.Info("Closing websocket stream", "connection_id", connID)
// 				if err := wspStream.CloseSend(); err != nil {
// 					slog.Error("Error closing workspace stream", "error", err)
// 				}
// 			}()

// 			// Error channel to capture goroutine errors
// 			errChan := make(chan error, 1)

// 			// Done channel to signal workspace goroutine completion
// 			doneChan := make(chan struct{})

// 			// Start goroutine to receive from workspace and forward to client
// 			go func() {
// 				defer close(doneChan) // Signal completion when the goroutine exits
// 				for {
// 					resp, err := wspStream.Recv()
// 					if err != nil {
// 						errChan <- fmt.Errorf("workspace recv failed: %w", err)
// 						return
// 					}
// 					slog.Info("Got response from workspace", "response", resp.Response)

// 					slog.Info("Forwarding response to WebSocket client", "response", resp.Response)
// 					if err := conn.WriteMessage(websocket.TextMessage, []byte(resp.Response)); err != nil {
// 						errChan <- fmt.Errorf("websocket send failed: %w", err)
// 						return
// 					}
// 					slog.Info("Response successfully sent to WebSocket client")
// 				}
// 			}()

// 			// Main loop: receive from client, send to workspace
// 			for {
// 				select {
// 				case <-ctx.Done():
// 					slog.Info("Context canceled, shutting down websocket")
// 					return
// 				case err := <-errChan:
// 					slog.Error("Error in websocket communication", "error", err)
// 					return
// 				case <-doneChan:
// 					slog.Info("Workspace receiver goroutine exited unexpectedly")
// 					return
// 				default:
// 					_, message, err := conn.ReadMessage()
// 					if err != nil {
// 						if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
// 							slog.Error("Websocket read error", "error", err)
// 						}
// 						return
// 					}

// 					slog.Info("Forwarding message to workspace", "message", string(message))
// 					if err := wspStream.Send(&wsp.LinkRequest{
// 						Request: string(message),
// 					}); err != nil {
// 						slog.Error("Error sending message to workspace", "error", err)
// 						return
// 					}
// 				}
// 			}

// 		} else {
// 			// Handle regular HTTP request
// 			w.WriteHeader(http.StatusOK)
// 			relayRouter.ServeHTTP(w, r)
// 			slog.Info("Served a regular HTTP request")
// 		}
// 	}
// }

func (l *LinkServer) ChatStream(stream ChatService_ChatStreamServer) (err error) {
	// Generate a unique connection ID for logging
	connID := fmt.Sprintf("conn_%d", time.Now().UnixNano())
	slog.Info("New client connected", "connection_id", connID)
	wspClient := wsp.NewWorkspaceServiceClient(l.WspConn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for Ctrl+C (SIGINT/SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Shutdown signal received")
		cancel() // triggers context cancellation
	}()
	wspStream, err := wspClient.LinkStream(ctx)

	if err != nil {
		slog.Error("Failed to create new stream")
	}

	defer func() {
		slog.Info("Closing stream", "connection_id", connID)
		// Don't send closing message if the connection is already closed
		if err == nil {
			if sendErr := stream.Send(&ChatResponse{Response: "Connection closed by server"}); sendErr != nil {
				slog.Info("Could not send closing message",
					"connection_id", connID,
					"error", sendErr)
			}
		}
	}()

	// Error channel to capture goroutine errors
	errChan := make(chan error, 1)

	// Done channel to signal workspace goroutine completion
	doneChan := make(chan struct{})

	// Start goroutine to receive from workspace and forward to client
	go func() {
		defer close(doneChan) // Signal completion when the goroutine exits
		for {
			resp, err := wspStream.Recv()
			if err != nil {
				errChan <- fmt.Errorf("workspace recv failed: %w", err)
				return
			}
			slog.Info("Got response from workspace")

			if err := stream.Send(&ChatResponse{
				Response: resp.Response,
				Trace:    resp.Trace,
			}); err != nil {
				errChan <- fmt.Errorf("client send failed: %w", err)
				return
			}
		}
	}()

	// Main loop: receive from client, send to workspace
	for {
		select {
		case <-ctx.Done():
			slog.Info("Context canceled, shutting down")
			return nil
		case err := <-errChan:
			return err
		case <-doneChan:
			return fmt.Errorf("workspace receiver goroutine exited unexpectedly")
		default:
			packet, err := stream.Recv()
			if err == io.EOF {
				slog.Info("Client closed connection")
				return nil
			}
			if err != nil {
				return fmt.Errorf("client recv error: %w", err)
			}

			if err := wspStream.Send(&wsp.LinkRequest{
				Request: packet.Request,
			}); err != nil {
				return fmt.Errorf("workspace send error: %w", err)
			}
		}
	}

}
