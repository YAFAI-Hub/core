package providers

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	//"os"
)

func GetProvider(name string) GenAIProvider {
	switch name {
	case "groq":
		return GroqProvider{Host: os.Getenv("GROQ_HOST")}
	case "ollama":
		return OllamaProvider{Host: os.Getenv("GROQ_HOST")}
	default:
		slog.Info("Unknown provider, falling back to groq")
		return GroqProvider{Host: os.Getenv("GROQ_HOST")}
	}
}

type GenAIProvider interface {
	Init() *http.Client
	Generate(ctx context.Context, client *http.Client, req GenAIProviderRequest) (*GenAIProviderResponse, error)
	//GenerateStream(ctx context.Context, client *http.Client, req GenAIProviderRequest) <-chan []byte @Todo
	Close(client *http.Client)
}
