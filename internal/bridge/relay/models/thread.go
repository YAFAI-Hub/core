package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
	Source    string    `json:"source" bson:"source"`
	Content   string    `json:"content" bson:"content"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
	Media     []string  `json:"media,omitempty" bson:"media,omitempty"`
}

type Thread struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	Name        string             `json:"name" bson:"name"`
	WorkspaceID primitive.ObjectID `json:"workspace_id" bson:"workspace_id"`
	Messages    []Message          `json:"messages" bson:"messages"`
	Status      string             `json:"status" bson:"status"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// BeforeCreate sets creation and update timestamps
func (t *Thread) BeforeCreate() {
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	// Set default status if not provided
	if t.Status == "" {
		t.Status = "active"
	}
}

// BeforeUpdate updates the UpdatedAt timestamp
func (t *Thread) BeforeUpdate() {
	t.UpdatedAt = time.Now()
}
