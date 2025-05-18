package repositories

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"yafai/internal/bridge/relay/models"
)

type WorkspaceRepository struct {
	collection *mongo.Collection
}

func NewWorkspaceRepository(db *mongo.Database) *WorkspaceRepository {
	return &WorkspaceRepository{
		collection: db.Collection("workspaces"),
	}
}

func (r *WorkspaceRepository) Create(ctx context.Context, workspace *models.Workspace) error {
	result, err := r.collection.InsertOne(ctx, workspace)
	if err != nil {
		return err
	}

	workspace.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *WorkspaceRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Workspace, error) {
	var workspace models.Workspace
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&workspace)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &workspace, nil
}

func (r *WorkspaceRepository) Update(ctx context.Context, workspace *models.Workspace) error {
	workspace.BeforeUpdate()

	update := bson.M{
		"$set": workspace,
	}

	_, err := r.collection.UpdateByID(ctx, workspace.ID, update)
	return err
}

func (r *WorkspaceRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *WorkspaceRepository) List(ctx context.Context, limit, offset int) ([]models.Workspace, error) {
	findOptions := options.Find()
	findOptions.SetLimit(int64(limit))
	findOptions.SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var workspaces []models.Workspace
	if err = cursor.All(ctx, &workspaces); err != nil {
		return nil, err
	}

	return workspaces, nil
}
