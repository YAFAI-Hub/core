package validation

import (
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ValidationMiddleware creates a middleware for request validation
func ValidationMiddleware(validate *validator.Validate, schema interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log request details for debugging
		logRequestDetails(c)

		// Create a new instance of the schema
		schemaType := reflect.TypeOf(schema)
		if schemaType.Kind() == reflect.Ptr {
			schemaType = schemaType.Elem()
		}

		// Create a new pointer to the schema type
		payloadPtr := reflect.New(schemaType).Interface()

		// Read the request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Failed to read request body",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		// Log the raw body for debugging
		log.Printf("Raw Request Body: %s", string(body))

		// Recreate the request body reader as it can only be read once
		c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

		// Bind JSON to the schema
		if err := c.ShouldBindJSON(payloadPtr); err != nil {
			// Handle specific error cases
			log.Printf("Binding Error: %v", err)

			switch {
			case err == io.EOF:
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Request body is empty",
					"details": "No data provided in the request body",
				})
			case strings.Contains(err.Error(), "cannot unmarshal"):
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid JSON format",
					"details": err.Error(),
				})
			default:
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid request payload",
					"details": err.Error(),
				})
			}
			c.Abort()
			return
		}

		// Validate the payload
		if err := validate.Struct(payloadPtr); err != nil {
			// Check if the error is a validation error
			validationErrors, ok := err.(validator.ValidationErrors)
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Validation failed",
				})
				c.Abort()
				return
			}

			// Collect and format validation errors
			var errors []string
			for _, e := range validationErrors {
				errors = append(errors, formatValidationError(e))
			}

			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Validation failed",
				"details": errors,
			})
			c.Abort()
			return
		}

		// Store the validated payload in the context
		c.Set("validatedPayload", payloadPtr)
		c.Next()
	}
}

// logRequestDetails logs details about the incoming request
func logRequestDetails(c *gin.Context) {
	log.Printf("Request Method: %s", c.Request.Method)
	log.Printf("Request URL: %s", c.Request.URL.String())

	// Log headers
	for key, values := range c.Request.Header {
		for _, value := range values {
			log.Printf("Header - %s: %s", key, value)
		}
	}
}

// formatValidationError creates a human-readable validation error message
func formatValidationError(e validator.FieldError) string {
	field := e.Field()

	// Convert camelCase to space-separated words
	field = splitCamelCase(field)

	switch e.Tag() {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email address"
	case "min":
		return field + " must be at least " + e.Param() + " characters"
	case "max":
		return field + " must be no more than " + e.Param() + " characters"
	case "oneof":
		return field + " must be one of: " + e.Param()
	default:
		return field + " is invalid"
	}
}

// splitCamelCase converts camelCase to space-separated words
func splitCamelCase(s string) string {
	var result []string
	var lastIdx int
	for i, r := range s {
		if i > 0 && isUpper(r) {
			result = append(result, s[lastIdx:i])
			lastIdx = i
		}
	}
	result = append(result, s[lastIdx:])

	return strings.Join(result, " ")
}

// isUpper checks if a rune is uppercase
func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}
