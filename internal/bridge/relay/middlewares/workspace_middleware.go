package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// WebSocketClaims defines the structure for WebSocket-specific JWT claims
type WebSocketClaims struct {
	Subject   string `json:"sub"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// ValidateWebSocketToken verifies the JWT token
func ValidateWebSocketToken(tokenString string) (*WebSocketClaims, error) {
	// Parse token
	token, err := jwt.ParseWithClaims(
		tokenString,
		&WebSocketClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}

			// Return secret key
			return []byte(os.Getenv("JWT_SECRET")), nil
		},
	)

	// Check parsing errors
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenMalformed):
			return nil, errors.New("malformed token")
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, errors.New("token has expired")
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			return nil, errors.New("token not valid yet")
		default:
			return nil, err
		}
	}

	// Extract and type assert claims
	claims, ok := token.Claims.(*WebSocketClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Additional token type validation
	if claims.TokenType != "websocket" {
		return nil, errors.New("invalid token type")
	}

	return claims, nil
}

// GenerateWebSocketToken creates a token specifically for WebSocket connections
func GenerateWebSocketToken(userID string) (string, error) {
	// Create claims
	claims := WebSocketClaims{
		Subject:   userID,
		TokenType: "websocket",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "yafai_websocket_service",
			Subject:   userID,
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
