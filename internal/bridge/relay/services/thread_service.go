package services

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"yafai/internal/bridge/relay/dto"
	"yafai/internal/bridge/relay/models"
	"yafai/internal/bridge/relay/repositories"
)

type ThreadService struct {
	repo *repositories.ThreadRepository
}

func NewThreadService(repo *repositories.ThreadRepository) *ThreadService {
	return &ThreadService{repo: repo}
}

func (s *ThreadService) Create(ctx context.Context, createDTO *dto.ThreadCreateDTO) (*models.Thread, error) {
	thread, err := createDTO.ToModel()
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, thread); err != nil {
		return nil, err
	}

	return thread, nil
}

func (s *ThreadService) GetByID(ctx context.Context, id string) (*models.Thread, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid thread ID")
	}

	return s.repo.FindByID(ctx, objectID)
}

func (s *ThreadService) GetThreadsByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]models.Thread, error) {
	objectID, err := primitive.ObjectIDFromHex(workspaceID)
	if err != nil {
		return nil, errors.New("invalid workspace ID")
	}

	return s.repo.FindByWorkspace(ctx, objectID, limit, offset)
}

func (s *ThreadService) Update(ctx context.Context, id string, updateDTO *dto.ThreadUpdateDTO) (*models.Thread, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid thread ID")
	}

	thread, err := s.repo.FindByID(ctx, objectID)
	if err != nil || thread == nil {
		return nil, errors.New("thread not found")
	}

	// Update fields if provided
	if updateDTO.Name != nil {
		thread.Name = *updateDTO.Name
	}
	if updateDTO.Status != nil {
		thread.Status = *updateDTO.Status
	}
	if updateDTO.Messages != nil {
		// Convert MessageDTOs to Message models
		messages := make([]models.Message, len(*updateDTO.Messages))
		for i, msgDTO := range *updateDTO.Messages {
			messages[i] = models.Message{
				Source:    msgDTO.Source,
				Content:   msgDTO.Content,
				Timestamp: msgDTO.Timestamp,
				Media:     msgDTO.Media,
			}

			// Set timestamp to current time if not provided
			if messages[i].Timestamp.IsZero() {
				messages[i].Timestamp = time.Now()
			}
		}
		thread.Messages = messages
	}

	if err := s.repo.Update(ctx, thread); err != nil {
		return nil, err
	}

	return thread, nil
}

func (s *ThreadService) AddMessage(ctx context.Context, threadID string, messageDTO *dto.MessageDTO) error {
	objectID, err := primitive.ObjectIDFromHex(threadID)
	if err != nil {
		return errors.New("invalid thread ID")
	}

	message := models.Message{
		Source:    messageDTO.Source,
		Content:   messageDTO.Content,
		Timestamp: time.Now(),
		Media:     messageDTO.Media,
	}

	return s.repo.AddMessage(ctx, objectID, message)
}

func (s *ThreadService) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid thread ID")
	}

	return s.repo.Delete(ctx, objectID)
}
