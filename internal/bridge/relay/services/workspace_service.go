package services

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"yafai/internal/bridge/relay/dto"
	"yafai/internal/bridge/relay/models"
	"yafai/internal/bridge/relay/repositories"
)

type WorkspaceService struct {
	repo *repositories.WorkspaceRepository
}

func NewWorkspaceService(repo *repositories.WorkspaceRepository) *WorkspaceService {
	return &WorkspaceService{repo: repo}
}

func (s *WorkspaceService) Create(ctx context.Context, createDTO *dto.WorkspaceCreateDTO) (*models.Workspace, error) {
	workspace := createDTO.ToModel()

	if err := s.repo.Create(ctx, workspace); err != nil {
		return nil, err
	}

	return workspace, nil
}

func (s *WorkspaceService) GetByID(ctx context.Context, id string) (*models.Workspace, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid workspace ID")
	}

	return s.repo.FindByID(ctx, objectID)
}

func (s *WorkspaceService) Update(ctx context.Context, id string, updateDTO *dto.WorkspaceUpdateDTO) (*models.Workspace, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid workspace ID")
	}

	workspace, err := s.repo.FindByID(ctx, objectID)
	if err != nil || workspace == nil {
		return nil, errors.New("workspace not found")
	}

	// Update fields if provided
	if updateDTO.Name != nil {
		workspace.Name = *updateDTO.Name
	}

	// Update Orchestrator if provided
	if updateDTO.Orchestrator != nil {
		// Implement orchestrator update logic
	}

	// Update Team if provided
	if updateDTO.Team != nil {
		// Implement team update logic
	}

	if err := s.repo.Update(ctx, workspace); err != nil {
		return nil, err
	}

	return workspace, nil
}

func (s *WorkspaceService) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid workspace ID")
	}

	return s.repo.Delete(ctx, objectID)
}

func (s *WorkspaceService) List(ctx context.Context, limit, offset int) ([]models.Workspace, error) {
	return s.repo.List(ctx, limit, offset)
}
