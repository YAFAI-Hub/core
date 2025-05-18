package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Agent struct {
	Name         string `json:"name" bson:"name"`
	Capabilities string `json:"capabilities" bson:"capabilities"`
	Description  string `json:"description" bson:"description"`
	Model        string `json:"model" bson:"model"`
	Provider     string `json:"provider" bson:"provider"`
	Goal         string `json:"goal" bson:"goal"`
	Status       string `json:"status" bson:"status"`
}

type Orchestrator struct {
	Name        string  `json:"name" bson:"name"`
	Description string  `json:"description" bson:"description"`
	Scope       string  `json:"scope" bson:"scope"`
	Model       string  `json:"model" bson:"model"`
	Provider    string  `json:"provider" bson:"provider"`
	Goal        string  `json:"goal" bson:"goal"`
	Team        []Agent `json:"team" bson:"team"`
}

type Workspace struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	Name         string             `json:"name" bson:"name"`
	Orchestrator Orchestrator       `json:"orchestrator" bson:"orchestrator"`
	Team         []Agent            `json:"team" bson:"team"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}

// BeforeCreate sets creation and update timestamps
func (w *Workspace) BeforeCreate() {
	w.CreatedAt = time.Now()
	w.UpdatedAt = time.Now()
}

// BeforeUpdate updates the UpdatedAt timestamp
func (w *Workspace) BeforeUpdate() {
	w.UpdatedAt = time.Now()
}
