package dto

import (
	"yafai/internal/bridge/relay/models"
)

type AgentDTO struct {
	Name         string `json:"name" validate:"required"`
	Capabilities string `json:"capabilities"`
	Description  string `json:"description"`
	Model        string `json:"model" validate:"required"`
	Provider     string `json:"provider" validate:"required"`
	Goal         string `json:"goal"`
	Status       string `json:"status"`
}

type OrchestratorDTO struct {
	Name        string     `json:"name" validate:"required"`
	Description string     `json:"description"`
	Scope       string     `json:"scope" validate:"required"`
	Model       string     `json:"model" validate:"required"`
	Provider    string     `json:"provider" validate:"required"`
	Goal        string     `json:"goal"`
	Team        []AgentDTO `json:"team"`
}

type WorkspaceCreateDTO struct {
	Name         string          `json:"name" validate:"required"`
	Scope        string          `json:"scope" validate:"required"`
	Orchestrator OrchestratorDTO `json:"orchestrator" validate:"required"`
	Team         []AgentDTO      `json:"team"`
}

type WorkspaceUpdateDTO struct {
	Name         *string          `json:"name,omitempty"`
	Scope        *string          `json:"scope,omitempty"`
	Orchestrator *OrchestratorDTO `json:"orchestrator,omitempty"`
	Team         *[]AgentDTO      `json:"team,omitempty"`
}

// ToModel converts DTO to Workspace model
func (dto *WorkspaceCreateDTO) ToModel() *models.Workspace {
	workspace := &models.Workspace{
		Name: dto.Name,
		Orchestrator: models.Orchestrator{
			Name:        dto.Orchestrator.Name,
			Description: dto.Orchestrator.Description,
			Scope:       dto.Orchestrator.Scope,
			Model:       dto.Orchestrator.Model,
			Provider:    dto.Orchestrator.Provider,
			Goal:        dto.Orchestrator.Goal,
		},
	}

	// Convert Agent DTOs
	workspace.Orchestrator.Team = make([]models.Agent, len(dto.Orchestrator.Team))
	for i, agentDTO := range dto.Orchestrator.Team {
		workspace.Orchestrator.Team[i] = models.Agent{
			Name:         agentDTO.Name,
			Capabilities: agentDTO.Capabilities,
			Description:  agentDTO.Description,
			Model:        agentDTO.Model,
			Provider:     agentDTO.Provider,
			Goal:         agentDTO.Goal,
			Status:       agentDTO.Status,
		}
	}

	workspace.Team = make([]models.Agent, len(dto.Team))
	for i, agentDTO := range dto.Team {
		workspace.Team[i] = models.Agent{
			Name:         agentDTO.Name,
			Capabilities: agentDTO.Capabilities,
			Description:  agentDTO.Description,
			Model:        agentDTO.Model,
			Provider:     agentDTO.Provider,
			Goal:         agentDTO.Goal,
			Status:       agentDTO.Status,
		}
	}

	workspace.BeforeCreate()
	return workspace
}
