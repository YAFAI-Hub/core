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

type GroqProvider struct {
	Host string
}

func (p GroqProvider) Init() *http.Client {
	client := http.Client{}
	return &client
}

func (p GroqProvider) Generate(ctx context.Context, client *http.Client, req GenAIProviderRequest) (*GenAIProviderResponse, error) {

	url := fmt.Sprintf("%s/v1/chat/completions", p.Host)
	token := fmt.Sprintf("Bearer %s", os.Getenv("GROQ_TOKEN"))

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
	req_obj.Header.Set("Authorization", token)

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

func (p GroqProvider) Close(client *http.Client) {
	client.CloseIdleConnections()
	return
}
