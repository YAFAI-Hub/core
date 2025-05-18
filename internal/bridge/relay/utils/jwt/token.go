package jwt

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func GenerateTokenPair(userID string) (*TokenPair, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))

	// Access Token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour).Unix(),
		"typ": "access",
	})

	// Refresh Token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"typ": "refresh",
	})

	// Sign tokens
	accessTokenString, err := accessToken.SignedString(secret)
	if err != nil {
		return nil, err
	}

	refreshTokenString, err := refreshToken.SignedString(secret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, nil
}

// In jwt utility
func ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}

		// Return secret key
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		return "", err
	}

	// Extract claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Validate token type (optional)
		if tokenType, ok := claims["typ"].(string); !ok || tokenType != "access" {
			return "", errors.New("invalid token type")
		}

		// Return user ID
		return claims["sub"].(string), nil
	}

	return "", errors.New("invalid token")
}

func RefreshAccessToken(refreshToken string) (string, error) {
	// Validate refresh token first
	userID, err := ValidateToken(refreshToken)
	if err != nil {
		return "", err
	}

	// Generate new access token
	tokenPair, err := GenerateTokenPair(userID)
	if err != nil {
		return "", err
	}

	return tokenPair.AccessToken, nil
}
