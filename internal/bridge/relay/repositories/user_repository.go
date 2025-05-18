package repositories

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"yafai/internal/bridge/relay/models"
)

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		collection: db.Collection("users"),
	}
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"username":   user.Username,
			"email":      user.Email,
			"last_login": user.LastLogin,
			"updated_at": user.UpdatedAt,
		},
	}

	_, err := r.collection.UpdateByID(ctx, user.ID, update)
	return err
}

// FindByUsername finds a user by username
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Create method with additional username uniqueness check
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	// Check if email already exists
	existingUser, err := r.FindByEmail(ctx, user.Email)
	if err != nil {
		return err
	}
	if existingUser != nil {
		return errors.New("user with this email already exists")
	}

	// Check if username already exists
	existingUsername, err := r.FindByUsername(ctx, user.Username)
	if err != nil {
		return err
	}
	if existingUsername != nil {
		return errors.New("username is already taken")
	}

	// Prepare user data
	if err := user.BeforeCreate(); err != nil {
		return err
	}

	// Insert user
	result, err := r.collection.InsertOne(ctx, bson.M{
		"username":      user.Username,
		"email":         user.Email,
		"password_hash": user.PasswordHash,
		"last_login":    user.LastLogin,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
	})
	if err != nil {
		return err
	}

	// Set the ID
	user.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}
