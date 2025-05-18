package repositories

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"yafai/internal/bridge/relay/models"
)

type ThreadRepository struct {
	collection *mongo.Collection
}

func NewThreadRepository(db *mongo.Database) *ThreadRepository {
	return &ThreadRepository{
		collection: db.Collection("threads"),
	}
}

func (r *ThreadRepository) Create(ctx context.Context, thread *models.Thread) error {
	result, err := r.collection.InsertOne(ctx, thread)
	if err != nil {
		return err
	}

	thread.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ThreadRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Thread, error) {
	var thread models.Thread
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&thread)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &thread, nil
}

func (r *ThreadRepository) FindByWorkspace(ctx context.Context, workspaceID primitive.ObjectID, limit, offset int) ([]models.Thread, error) {
	findOptions := options.Find()
	findOptions.SetLimit(int64(limit))
	findOptions.SetSkip(int64(offset))

	cursor, err := r.collection.Find(
		ctx,
		bson.M{"workspace_id": workspaceID},
		findOptions,
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var threads []models.Thread
	if err = cursor.All(ctx, &threads); err != nil {
		return nil, err
	}

	return threads, nil
}

func (r *ThreadRepository) Update(ctx context.Context, thread *models.Thread) error {
	thread.BeforeUpdate()

	update := bson.M{
		"$set": thread,
	}

	_, err := r.collection.UpdateByID(ctx, thread.ID, update)
	return err
}

func (r *ThreadRepository) AddMessage(ctx context.Context, threadID primitive.ObjectID, message models.Message) error {
	update := bson.M{
		"$push": bson.M{
			"messages": message,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	_, err := r.collection.UpdateByID(ctx, threadID, update)
	return err
}

func (r *ThreadRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
