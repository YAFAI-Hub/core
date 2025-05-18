package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"yafai/internal/bridge/relay/dto"
	"yafai/internal/bridge/relay/services"
)

type userHandler struct {
	service  *services.UserService
	validate *validator.Validate
}

func NewUserHandler(service *services.UserService, validate *validator.Validate) *userHandler {
	return &userHandler{
		service:  service,
		validate: validate,
	}
}

func (h *userHandler) Register(c *gin.Context) {
	// Log request details
	log.Printf("User Registration Request: %s", c.Request.URL.Path)

	// Retrieve the validated payload
	payload, exists := c.Get("validatedPayload")
	if !exists {
		log.Println("Error: Validation payload not found")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Validation payload not found",
		})
		return
	}

	// Type assert to the correct DTO type
	createDTO, ok := payload.(*dto.UserCreateDTO)
	if !ok {
		log.Println("Error: Invalid payload type")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid payload type",
		})
		return
	}

	// Log registration attempt (sanitize sensitive info)
	log.Printf("Registration Attempt - Username: %s, Email: %s",
		createDTO.Username,
		createDTO.Email,
	)

	// Proceed with registration
	user, err := h.service.Register(c.Request.Context(), createDTO)
	if err != nil {
		log.Printf("Registration Error: %v", err)

		// Detailed error handling
		switch {
		case strings.Contains(err.Error(), "already exists"):
			c.JSON(http.StatusConflict, gin.H{
				"error":   "User registration failed",
				"details": "User with this email already exists",
			})
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "User registration failed",
				"details": err.Error(),
			})
		}
		return
	}

	log.Printf("User Registered Successfully - ID: %s, Username: %s",
		user.ID.Hex(),
		user.Username,
	)

	c.JSON(http.StatusCreated, user.Sanitize())
}

func (h *userHandler) Login(c *gin.Context) {
	// Log request details
	log.Printf("User Login Request: %s", c.Request.URL.Path)

	// Retrieve the validated payload
	payload, exists := c.Get("validatedPayload")
	if !exists {
		log.Println("Error: Validation payload not found")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Validation payload not found",
		})
		return
	}

	// Type assert to the correct DTO type
	loginDTO, ok := payload.(*dto.UserLoginDTO)
	if !ok {
		log.Println("Error: Invalid payload type")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid payload type",
		})
		return
	}

	// Log login attempt (sanitize sensitive info)
	log.Printf("Login Attempt - Email: %s", loginDTO.Email)

	// Proceed with login
	user, tokens, err := h.service.Login(c.Request.Context(), loginDTO)
	if err != nil {
		log.Printf("Login Error: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Login failed",
			"details": err.Error(),
		})
		return
	}

	log.Printf("User Logged In Successfully - ID: %s, Email: %s",
		user.ID.Hex(),
		user.Email,
	)

	c.JSON(http.StatusOK, gin.H{
		"user":   user.Sanitize(),
		"tokens": tokens,
	})
}

func (h *userHandler) GetUser(c *gin.Context) {
	// Extract user ID from path
	userID := c.Param("id")

	// Log request details
	log.Printf("Get User Request - ID: %s", userID)

	// Retrieve user
	user, err := h.service.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		log.Printf("Get User Error: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "User not found",
			"details": err.Error(),
		})
		return
	}

	log.Printf("User Retrieved Successfully - ID: %s, Email: %s",
		user.ID.Hex(),
		user.Email,
	)

	c.JSON(http.StatusOK, user.Sanitize())
}

func (h *userHandler) UpdateUser(c *gin.Context) {
	// Extract user ID from path
	userID := c.Param("id")

	// Log request details
	log.Printf("Update User Request - ID: %s", userID)

	// Retrieve the validated payload
	payload, exists := c.Get("validatedPayload")
	if !exists {
		log.Println("Error: Validation payload not found")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Validation payload not found",
		})
		return
	}

	// Type assert to the correct DTO type
	updateDTO, ok := payload.(*dto.UserUpdateDTO)
	if !ok {
		log.Println("Error: Invalid payload type")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid payload type",
		})
		return
	}

	// Log update attempt
	log.Printf("Update User Attempt - ID: %s", userID)

	// Proceed with update
	user, err := h.service.UpdateUser(c.Request.Context(), userID, updateDTO)
	if err != nil {
		log.Printf("Update User Error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "User update failed",
			"details": err.Error(),
		})
		return
	}

	log.Printf("User Updated Successfully - ID: %s, Email: %s",
		user.ID.Hex(),
		user.Email,
	)

	c.JSON(http.StatusOK, user.Sanitize())
}

// Singleton instance
var UserHandler *userHandler
