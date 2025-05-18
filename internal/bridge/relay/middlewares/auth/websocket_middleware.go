package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

type WebSocketClaims struct {
	Subject   string `json:"sub"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

func ConvertToWebSocketToken(accessToken string) (string, error) {
	// Parse the existing token without verification
	token, _, err := new(jwt.Parser).ParseUnverified(accessToken, jwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %v", err)
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}

	// Create new claims with websocket type
	webSocketClaims := WebSocketClaims{
		Subject:   claims["sub"].(string),
		TokenType: "websocket",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Unix(int64(claims["exp"].(float64)), 0)),
			IssuedAt:  jwt.NewNumericDate(time.Unix(int64(claims["iat"].(float64)), 0)),
			Issuer:    claims["iss"].(string),
		},
	}

	// Create new token
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, webSocketClaims)

	// Sign the token (use your secret key)
	return newToken.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

// Modify ValidateWebSocketToken to be more flexible
func ValidateWebSocketToken(tokenString string) (*jwt.Token, error) {
	// Parse token without signature verification
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &WebSocketClaims{})
	if err != nil {
		return nil, fmt.Errorf("token parse error: %v", err)
	}

	// Extract claims
	claims, ok := token.Claims.(*WebSocketClaims)
	if !ok {
		// Try map claims for flexibility
		mapClaims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, errors.New("invalid token claims")
		}

		// Check and convert token type
		tokenType, ok := mapClaims["token_type"].(string)
		if !ok || tokenType != "websocket" {
			// Attempt to convert token
			convertedToken, err := ConvertToWebSocketToken(tokenString)
			if err != nil {
				return nil, errors.New("invalid or non-convertible token")
			}

			// Re-parse the converted token
			return ValidateWebSocketToken(convertedToken)
		}
	}

	// Manual expiration check
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return nil, errors.New("token has expired")
	}

	// Additional custom validation
	if claims.Subject == "" {
		return nil, errors.New("missing user ID")
	}

	return token, nil
}

func GenerateWebSocketToken(userID string) (string, error) {
	claims := WebSocketClaims{
		Subject:   userID,
		TokenType: "websocket",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "your_service",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

// ExtractWebSocketToken retrieves the token from various sources
func ExtractWebSocketToken(r *http.Request) (string, error) {
	// 1. Check query parameter
	token := r.URL.Query().Get("token")
	if token != "" {
		return token, nil
	}

	// 2. Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1], nil
		}
	}

	// 3. Check Sec-WebSocket-Protocol header (if applicable)
	token = r.Header.Get("Sec-WebSocket-Protocol")
	if token != "" {
		return token, nil
	}

	return "", errors.New("no token found")
}

// WebSocketAuthMiddleware handles authentication for WebSocket connections
func WebSocketAuthMiddleware(next func(w http.ResponseWriter, r *http.Request, userID string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract token
		token, err := ExtractWebSocketToken(r)
		if err != nil {
			slog.Error("Token extraction failed",
				"error", err,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// 2. Validate token
		parsedToken, err := ValidateWebSocketToken(token)
		if err != nil {
			slog.Error("Token validation failed",
				"error", err,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// 3. Extract claims
		claims, ok := parsedToken.Claims.(*WebSocketClaims)
		if !ok {
			slog.Error("Invalid token claims",
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "Unauthorized: invalid token claims", http.StatusUnauthorized)
			return
		}

		// 4. Call next handler with user ID
		next(w, r, claims.Subject)
	}
}

// CreateWebSocketUpgrader provides a secure WebSocket upgrader
func CreateWebSocketUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Implement strict origin checking
			origin := r.Header.Get("Origin")

			// Log origin for debugging
			slog.Info("WebSocket origin check",
				"origin", origin,
				"remote_addr", r.RemoteAddr,
			)

			return isValidOrigin(origin)
		},
	}
}

// isValidOrigin checks if the origin is allowed
func isValidOrigin(origin string) bool {
	// If no origin is provided, reject
	if origin == "" {
		slog.Warn("Empty origin detected")
		return false
	}

	// Implement origin validation logic
	allowedOrigins := []string{
		"https://yourdomain.com",
		"http://localhost:3000",
		"http://localhost:8080",
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		slog.Error("Failed to parse origin",
			"origin", origin,
			"error", err,
		)
		return false
	}

	for _, allowed := range allowedOrigins {
		parsedAllowed, err := url.Parse(allowed)
		if err != nil {
			slog.Error("Failed to parse allowed origin",
				"allowed_origin", allowed,
				"error", err,
			)
			continue
		}

		// More comprehensive origin matching
		if parsedOrigin.Scheme == parsedAllowed.Scheme &&
			parsedOrigin.Hostname() == parsedAllowed.Hostname() &&
			(parsedAllowed.Port() == "" || parsedOrigin.Port() == parsedAllowed.Port()) {
			return true
		}
	}

	slog.Warn("Origin not allowed",
		"origin", origin,
	)

	return false
}
