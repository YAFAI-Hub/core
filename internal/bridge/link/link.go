package link

import (
	"fmt"
	"io"
	"log/slog"
	"time"
	wsp "yafai/internal/bridge/wsp"
)

func (l *LinkServer) ChatStream(stream ChatService_ChatStreamServer) (err error) {
	// Generate a unique connection ID for logging
	connID := fmt.Sprintf("conn_%d", time.Now().UnixNano())
	slog.Info("New client connected", "connection_id", connID)

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
			resp, err := l.WspStream.Recv()
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

			if err := l.WspStream.Send(&wsp.LinkRequest{
				Request: packet.Request,
			}); err != nil {
				return fmt.Errorf("workspace send error: %w", err)
			}
		}
	}

}
