package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"yafai/internal/bridge/relay/dto"
	"yafai/internal/bridge/relay/services"
)

type workspaceHandler struct {
	service  *services.WorkspaceService
	validate *validator.Validate
}

func NewWorkspaceHandler(service *services.WorkspaceService, validate *validator.Validate) *workspaceHandler {
	return &workspaceHandler{
		service:  service,
		validate: validate,
	}
}

func (h *workspaceHandler) Create(c *gin.Context) {
	// Log request details
	log.Printf("Workspace Creation Request: %s", c.Request.URL.Path)

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
	createDTO, ok := payload.(*dto.WorkspaceCreateDTO)
	if !ok {
		log.Println("Error: Invalid payload type")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid payload type",
		})
		return
	}

	// Log creation attempt
	log.Printf("Workspace Creation Attempt - Name: %s, Scope: %s",
		createDTO.Name,
		createDTO.Scope,
	)

	// Proceed with creation
	workspace, err := h.service.Create(c.Request.Context(), createDTO)
	if err != nil {
		log.Printf("Workspace Creation Error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Workspace creation failed",
			"details": err.Error(),
		})
		return
	}

	log.Printf("Workspace Created Successfully - ID: %s, Name: %s",
		workspace.ID.Hex(),
		workspace.Name,
	)

	c.JSON(http.StatusCreated, workspace)
}

func (h *workspaceHandler) GetByID(c *gin.Context) {
	// Extract workspace ID from path
	workspaceID := c.Param("id")

	// Log request details
	log.Printf("Get Workspace Request - ID: %s", workspaceID)

	// Retrieve workspace
	workspace, err := h.service.GetByID(c.Request.Context(), workspaceID)
	if err != nil {
		log.Printf("Get Workspace Error: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Workspace not found",
			"details": err.Error(),
		})
		return
	}

	log.Printf("Workspace Retrieved Successfully - ID: %s, Name: %s",
		workspace.ID.Hex(),
		workspace.Name,
	)

	c.JSON(http.StatusOK, workspace)
}

func (h *workspaceHandler) Update(c *gin.Context) {
	// Extract workspace ID from path
	workspaceID := c.Param("id")

	// Log request details
	log.Printf("Update Workspace Request - ID: %s", workspaceID)

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
	updateDTO, ok := payload.(*dto.WorkspaceUpdateDTO)
	if !ok {
		log.Println("Error: Invalid payload type")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid payload type",
		})
		return
	}

	// Log update attempt
	log.Printf("Update Workspace Attempt - ID: %s", workspaceID)

	// Proceed with update
	workspace, err := h.service.Update(c.Request.Context(), workspaceID, updateDTO)
	if err != nil {
		log.Printf("Update Workspace Error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Workspace update failed",
			"details": err.Error(),
		})
		return
	}

	log.Printf("Workspace Updated Successfully - ID: %s, Name: %s",
		workspace.ID.Hex(),
		workspace.Name,
	)

	c.JSON(http.StatusOK, workspace)
}

func (h *workspaceHandler) Delete(c *gin.Context) {
	// Extract workspace ID from path
	workspaceID := c.Param("id")

	// Log request details
	log.Printf("Delete Workspace Request - ID: %s", workspaceID)

	// Proceed with deletion
	err := h.service.Delete(c.Request.Context(), workspaceID)
	if err != nil {
		log.Printf("Delete Workspace Error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Workspace deletion failed",
			"details": err.Error(),
		})
		return
	}

	log.Printf("Workspace Deleted Successfully - ID: %s", workspaceID)

	c.JSON(http.StatusNoContent, nil)
}

func (h *workspaceHandler) List(c *gin.Context) {
	// Extract pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Log request details
	log.Printf("List Workspaces Request - Limit: %d, Offset: %d", limit, offset)

	// Retrieve workspaces
	workspaces, err := h.service.List(c.Request.Context(), limit, offset)
	if err != nil {
		log.Printf("List Workspaces Error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to retrieve workspaces",
			"details": err.Error(),
		})
		return
	}

	log.Printf("Workspaces Retrieved Successfully - Count: %d", len(workspaces))

	c.JSON(http.StatusOK, workspaces)
}

// Singleton instance
var WorkspaceHandler *workspaceHandler
