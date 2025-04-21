package config

import (
	"log/slog"
	"os"
	"yafai/internal/nexus/workspace"

	"gopkg.in/yaml.v3"
)

func GetAvailableConfigs(path string) ([]string, error) {
	var configs []string
	files, err := os.ReadDir(path)
	if err != nil {
		slog.Error("Failed to read directory", "error", err)
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			configs = append(configs, file.Name())
		}
	}

	return configs, nil
}

func NewWorkspace(path string) *workspace.Workspace {
	wsp := ParseConfig(path)
	return wsp
}

func ParseConfig(path string) *workspace.Workspace {

	data, err := os.ReadFile(path)

	if err != nil {
		slog.Error("here!!!", err.Error(), nil)
	}

	var config WorkspaceConfig
	err = yaml.Unmarshal(data, &config)

	if err != nil {
		slog.Info("YAML parsing error!!!!!!!!!!!!!!!")
		slog.Error(err.Error())
	}

	for _, member := range config.Orchestrator.Team {
		config.Planner.Agents = append(config.Planner.Agents, member)
	}
	// planner := &executors.YafaiPlanner{Agents: config.Team, Model: config.Planner.Model }

	workspace := workspace.Workspace{
		Name:         config.Name,
		Scope:        config.Scope,
		Planner:      &config.Planner,
		Orchestrator: &config.Orchestrator,
		Integrations: config.Integrations,
		VectorStore:  config.VectorStore,
		Bridge:       config.Bridge,
	}

	return &workspace
}
