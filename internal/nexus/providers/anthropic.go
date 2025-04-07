package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

type AnthropicProvider struct {
	Host string
}

func (p AnthropicProvider) Init() *http.Client {
	client := http.Client{}
	return &client
}

func (p AnthropicProvider) Generate(ctx context.Context, client *http.Client, req GenAIProviderRequest) (*GenAIProviderResponse, error) {

	url := fmt.Sprintf("%s/v1/messages", p.Host)
	token := os.Getenv("ANTHROPIC_TOKEN")

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(req)
	if err != nil {
		slog.Info("%s", err)
	}

	//slog.Info(req)
	req_obj, err := http.NewRequest(http.MethodPost, url, &buf)

	if err != nil {
		panic(fmt.Sprintf("Error creating request:", err))

	}
	req_obj.Header.Set("Content-Type", "application/json")
	req_obj.Header.Set("x-api-key", token)

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
	//slog.Info(result.Message.ToolCall)
	return &result, nil

}

func (p AnthropicProvider) Close(client *http.Client) {
	client.CloseIdleConnections()
	return
}
