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

		slog.Info("Received packet",
			"connection_id", connID, "message", packet)

		// Create channels for error handling
		errChan := make(chan error, 1)
		done := make(chan struct{})

		// Handle client -> workspace communication
		if err := l.WspStream.Send(&wsp.LinkRequest{
			Request: packet.Request,
		}); err != nil {
			close(done)
			return err
		}

		// Handle workspace -> client communication
		go func() {
			for {
				resp, err := l.WspStream.Recv()
				if err != nil {
					errChan <- err
					return
				}

				if err := stream.Send(&ChatResponse{
					Response: resp.Response,
					Trace:    resp.Trace,
				}); err != nil {
					errChan <- err
					return
				}
			}
		}()

		// Handle client -> workspace communication in main loop

	}
}
