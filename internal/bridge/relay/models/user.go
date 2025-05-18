package models

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username     string             `bson:"username" json:"username"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"password_hash" json:"-"`
	LastLogin    time.Time          `bson:"last_login" json:"last_login"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// BeforeCreate prepares user data before insertion
func (u *User) BeforeCreate() error {
	// Validate required fields
	if u.Username == "" {
		return errors.New("username is required")
	}
	if u.Email == "" {
		return errors.New("email is required")
	}
	if u.PasswordHash == "" {
		return errors.New("password is required")
	}

	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	u.LastLogin = time.Now()

	return nil
}

// SetPassword hashes and sets the password
func (u *User) SetPassword(password string) error {
	// Generate password hash
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.PasswordHash = string(hashedPassword)
	return nil
}

// CheckPassword verifies the provided password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// Sanitize removes sensitive information
func (u *User) Sanitize() *User {
	return &User{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		LastLogin: u.LastLogin,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
