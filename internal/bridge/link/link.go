package link

import (
	context "context"
	"fmt"
	"io"
	"log/slog"
	"time"
	wsp "yafai/internal/bridge/wsp"

	grpc "google.golang.org/grpc"
)

func NewLinkServer(workspaceServerAddress string) (*LinkServer, error) {
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

func (l *LinkServer) ChatStream(stream ChatService_ChatStreamServer) (err error) {
	// Generate a unique connection ID for logging
	connID := fmt.Sprintf("conn_%d", time.Now().UnixNano())
	slog.Info("New client connected", "connection_id", connID)
	wspClient := wsp.NewWorkspaceServiceClient(l.WspConn)
	wspStream, err := wspClient.LinkStream(context.Background())

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
		case err := <-errChan:
			return err // Workspace goroutine error
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
