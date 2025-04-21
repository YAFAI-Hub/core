package config

import (
	"yafai/internal/nexus/executors"
)

type WorkspaceConfig struct {
	Name         string                      `yaml:"name"`
	Scope        string                      `yaml:"scope"`
	Planner      executors.YafaiPlanner      `yaml:"planner,omitempty"`
	Orchestrator executors.YafaiOrchestrator `yaml:"orchestrator,omitempty"`
	Integrations []string                    `yaml:"integrations,omitempty"`
	VectorStore  string                      `yaml:"vector_store,omitempty"`
	Bridge       string                      `yaml:"bridge"`
}
