package workspace

import (
	"sync"
	"yafai/internal/nexus/executors"
)

type Workspace struct {
	Name         string                       `json:"name" yaml:"name"`
	Scope        string                       `json:"scope" yaml:"scope"`
	Planner      *executors.YafaiPlanner      `json:"planner" yaml:"planner"`
	Orchestrator *executors.YafaiOrchestrator `json:"orchestrator,omitempty" yaml:"orchestrator,omitempty"`
	Integrations []string                     `json:"integrations,omitempty" yaml:"integrations,omitempty"`
	VectorStore  string                       `json:"vector_store,omitempty" yaml:"vector_store,omitempty"`
	Bridge       string                       `json:"bridge" yaml:"bridge"`
	ListenerPool sync.WaitGroup               `json:"pool" yaml:"pool"`
}
