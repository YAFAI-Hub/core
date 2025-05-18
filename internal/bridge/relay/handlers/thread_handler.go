package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"yafai/internal/bridge/relay/dto"
	"yafai/internal/bridge/relay/services"
)

type threadHandler struct {
	service  *services.ThreadService
	validate *validator.Validate
}

func NewThreadHandler(service *services.ThreadService, validate *validator.Validate) *threadHandler {
	return &threadHandler{
		service:  service,
		validate: validator.New(),
	}
}

func (h *threadHandler) Create(c *gin.Context) {
	var createDTO dto.ThreadCreateDTO
	if err := c.ShouldBindJSON(&createDTO); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate DTO
	if err := h.validate.Struct(createDTO); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	thread, err := h.service.Create(c.Request.Context(), &createDTO)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, thread)
}

func (h *threadHandler) GetByID(c *gin.Context) {
	threadID := c.Param("id")

	thread, err := h.service.GetByID(c.Request.Context(), threadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, thread)
}

func (h *threadHandler) GetThreadsByWorkspace(c *gin.Context) {
	workspaceID := c.Param("workspace_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	threads, err := h.service.GetThreadsByWorkspace(c.Request.Context(), workspaceID, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, threads)
}

func (h *threadHandler) Update(c *gin.Context) {
	threadID := c.Param("id")

	var updateDTO dto.ThreadUpdateDTO
	if err := c.ShouldBindJSON(&updateDTO); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate DTO
	if err := h.validate.Struct(updateDTO); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	thread, err := h.service.Update(c.Request.Context(), threadID, &updateDTO)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, thread)
}

func (h *threadHandler) AddMessage(c *gin.Context) {
	threadID := c.Param("id")

	var messageDTO dto.MessageDTO
	if err := c.ShouldBindJSON(&messageDTO); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate DTO
	if err := h.validate.Struct(messageDTO); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AddMessage(c.Request.Context(), threadID, &messageDTO); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message added successfully"})
}

func (h *threadHandler) Delete(c *gin.Context) {
	threadID := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), threadID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Singleton instance
var ThreadHandler *threadHandler
