package services

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"yafai/internal/bridge/relay/dto"
	"yafai/internal/bridge/relay/models"
	"yafai/internal/bridge/relay/repositories"
	"yafai/internal/bridge/relay/utils/jwt"
)

type UserService struct {
	repo *repositories.UserRepository
}

func NewUserService(repo *repositories.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(ctx context.Context, createDTO *dto.UserCreateDTO) (*models.User, error) {
	// Validate username length
	if len(createDTO.Username) < 3 || len(createDTO.Username) > 50 {
		return nil, errors.New("username must be between 3 and 50 characters")
	}

	// Check for invalid characters in username (optional)
	if !isValidUsername(createDTO.Username) {
		return nil, errors.New("username can only contain letters, numbers, and underscores")
	}

	// Create new user model
	user := &models.User{
		Username: createDTO.Username,
		Email:    createDTO.Email,
	}

	// Set password with hashing
	if err := user.SetPassword(createDTO.Password); err != nil {
		return nil, err
	}

	// Create user in repository
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// isValidUsername checks if username contains only valid characters
func isValidUsername(username string) bool {
	for _, char := range username {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return false
		}
	}
	return true
}
func (s *UserService) Login(ctx context.Context, loginDTO *dto.UserLoginDTO) (*models.User, *jwt.TokenPair, error) {
	// Find user by email
	user, err := s.repo.FindByEmail(ctx, loginDTO.Email)
	if err != nil || user == nil {
		return nil, nil, errors.New("invalid credentials")
	}

	// Verify password
	if !user.CheckPassword(loginDTO.Password) {
		return nil, nil, errors.New("invalid credentials")
	}

	// Update last login
	user.LastLogin = time.Now()
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, nil, err
	}

	// Generate tokens
	tokens, err := jwt.GenerateTokenPair(user.ID.Hex())
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}
	return s.repo.FindByID(ctx, objectID)
}

func (s *UserService) UpdateUser(ctx context.Context, id string, updateDTO *dto.UserUpdateDTO) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	user, err := s.repo.FindByID(ctx, objectID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	// Update fields if provided
	if updateDTO.Username != nil {
		user.Username = *updateDTO.Username
	}
	if updateDTO.Email != nil {
		user.Email = *updateDTO.Email
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
