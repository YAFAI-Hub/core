package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type OllamaProvider struct {
	Host string
}

func (p OllamaProvider) Init() *http.Client {
	client := http.Client{}
	slog.Info(fmt.Sprintf("Provider client initialised on %s", p.Host))
	return &client

}

func (p OllamaProvider) Generate(ctx context.Context, client *http.Client, req GenAIProviderRequest) (*GenAIProviderResponse, error) {

	url := fmt.Sprintf("%s/v1/chat/completions", p.Host)

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(req)
	if err != nil {
		slog.Info("%s", err)
	}
	req_obj, err := http.NewRequest(http.MethodPost, url, &buf)

	if err != nil {
		slog.Error(err.Error())
	}
	resp, err := client.Do(req_obj)
	if err != nil {
		slog.Error(err.Error())
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error(err.Error())
	}
	// Unmarshal the response body into the struct
	var result GenAIProviderResponse

	err = json.Unmarshal(bytes, &result)
	if err != nil {
		slog.Error("Error unmarshaling response")
	}
	return &result, nil

}

func (p OllamaProvider) Close(client *http.Client) {
	client.CloseIdleConnections()
	slog.Info("Provider client released.")
}
