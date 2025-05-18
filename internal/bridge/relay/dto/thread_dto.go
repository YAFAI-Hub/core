package dto

import (
	"time"

	"yafai/internal/bridge/relay/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MessageDTO struct {
	Source    string    `json:"source" validate:"required"`
	Content   string    `json:"content" validate:"required"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	Media     []string  `json:"media,omitempty"`
}

type ThreadCreateDTO struct {
	Name        string       `json:"name" validate:"required"`
	WorkspaceID string       `json:"workspace_id" validate:"required"`
	Messages    []MessageDTO `json:"messages,omitempty"`
	Status      string       `json:"status,omitempty" validate:"omitempty,oneof=active archived"`
}

type ThreadUpdateDTO struct {
	Name     *string       `json:"name,omitempty"`
	Messages *[]MessageDTO `json:"messages,omitempty"`
	Status   *string       `json:"status,omitempty" validate:"omitempty,oneof=active archived"`
}

// ToModel converts CreateDTO to Thread model
func (dto *ThreadCreateDTO) ToModel() (*models.Thread, error) {
	// Convert workspace_id string to ObjectID
	workspaceID, err := primitive.ObjectIDFromHex(dto.WorkspaceID)
	if err != nil {
		return nil, err
	}

	// Convert MessageDTOs to Message models
	messages := make([]models.Message, len(dto.Messages))
	for i, msgDTO := range dto.Messages {
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

	thread := &models.Thread{
		Name:        dto.Name,
		WorkspaceID: workspaceID,
		Messages:    messages,
		Status:      dto.Status,
	}

	thread.BeforeCreate()
	return thread, nil
}
