package relay

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"

	"yafai/internal/bridge/relay/dto"
	"yafai/internal/bridge/relay/handlers"
	"yafai/internal/bridge/relay/middlewares/auth"
	"yafai/internal/bridge/relay/middlewares/validation"
	"yafai/internal/bridge/relay/repositories"
	"yafai/internal/bridge/relay/services"
	"yafai/internal/bridge/wsp"
)

// Config represents the configuration for setting up the relay
type Config struct {
	MongoURI        string
	DatabaseName    string
	WorkspaceAddr   string
	RelayServerAddr string
}

// WebSocket upgrader with origin checking
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Implement strict origin checking if needed
		return true
	},
}

// Assuming you have a custom claims struct in your auth package
func HandleWebSocketConnection(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Missing authentication token",
		})
		return
	}

	// 2. Validate token
	parsedToken, err := auth.ValidateWebSocketToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Invalid token",
			"details": err.Error(),
		})
		return
	}

	// 3. Extract claims
	claims, ok := parsedToken.Claims.(*auth.WebSocketClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token claims",
		})
		return
	}

	// 4. Get user ID
	userID := claims.Subject

	// 3. Upgrade WebSocket connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("Failed to upgrade WebSocket",
			"error", err,
			"user_id", userID,
		)
		return
	}
	defer conn.Close()

	// 4. Extract thread from query parameter
	thread := c.Query("thread")
	if thread == "" {
		thread = "default"
	}

	// 5. Log successful connection
	slog.Info("WebSocket connection established",
		"user_id", userID,
		"thread", thread,
	)

	// 6. Main WebSocket message handling loop
	for {
		// Read incoming message
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			// Check for specific WebSocket close errors
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				slog.Error("WebSocket unexpected close",
					"error", err,
					"user_id", userID,
				)
			}
			break
		}

		// Process the message
		slog.Info("Received WebSocket message",
			"user_id", userID,
			"thread", thread,
			"message", string(message),
		)

		// Echo message back (optional)
		err = conn.WriteMessage(messageType, message)
		if err != nil {
			slog.Error("WebSocket write error",
				"error", err,
				"user_id", userID,
			)
			break
		}
	}
}

// SetupRelay sets up and returns a configured Gin router
func SetupRelay(cfg Config) (*gin.Engine, error) {
	// Create global validator
	validate := validator.New()
	// In relay setup or validation configuration
	validate.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		username := fl.Field().String()

		// Check length
		if len(username) < 3 || len(username) > 50 {
			return false
		}

		// Check for valid characters
		for _, char := range username {
			if !((char >= 'a' && char <= 'z') ||
				(char >= 'A' && char <= 'Z') ||
				(char >= '0' && char <= '9') ||
				char == '_') {
				return false
			}
		}

		return true
	})

	// Setup MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup MongoDB client
	clientOptions := options.Client().ApplyURI(cfg.MongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Ping the database
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	// Get database
	db := client.Database(cfg.DatabaseName)

	//Establish gRPC connection to the workspace server
	grpcConn, err := grpc.Dial(cfg.WorkspaceAddr, grpc.WithInsecure())
	if err != nil {
		slog.Error("Failed to dial workspace server", "error", err)
	}

	defer grpcConn.Close()
	// Setup Gin router
	// Setup Gin router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Add this to help with debugging JSON parsing
	router.Use(func(c *gin.Context) {
		if c.ContentType() != "application/json" {
			log.Printf("Unexpected Content-Type: %s", c.ContentType())
		}
		c.Next()
	})

	// Health check route
	router.GET("/health", func(c *gin.Context) {
		status := "healthy"

		// Check database connectivity
		err := db.Client().Ping(context.Background(), nil)
		if err != nil {
			status = "unhealthy"
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": status,
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": status,
		})
	})

	// Setup repositories
	userRepo := repositories.NewUserRepository(db)
	workspaceRepo := repositories.NewWorkspaceRepository(db)
	threadRepo := repositories.NewThreadRepository(db)

	// Setup services
	userService := services.NewUserService(userRepo)
	workspaceService := services.NewWorkspaceService(workspaceRepo)
	threadService := services.NewThreadService(threadRepo)

	// Setup handlers with global validator
	handlers.UserHandler = handlers.NewUserHandler(userService, validate)
	handlers.WorkspaceHandler = handlers.NewWorkspaceHandler(workspaceService, validate)
	handlers.ThreadHandler = handlers.NewThreadHandler(threadService, validate)

	workspaceClient := wsp.NewWorkspaceServiceClient(grpcConn)

	// Initialize WebSocket handler
	handlers.WebSocketHandler = handlers.NewChatWebSocketHandler(workspaceClient)

	// Configure routes
	v1 := router.Group("/api/v1")
	{

		// Chat WebSocket route with authentication middleware
		v1.GET("/chat/ws", HandleWebSocketConnection)
		// User routes
		users := v1.Group("/users")
		{
			users.POST("/register",
				validation.ValidationMiddleware(validate, dto.UserCreateDTO{}),
				handlers.UserHandler.Register,
			)
			users.POST("/login",
				validation.ValidationMiddleware(validate, dto.UserLoginDTO{}),
				handlers.UserHandler.Login,
			)
			users.GET("/:id",
				auth.JWTMiddleware(),
				handlers.UserHandler.GetUser,
			)
			users.PATCH("/:id",
				auth.JWTMiddleware(),
				validation.ValidationMiddleware(validate, dto.UserUpdateDTO{}),
				handlers.UserHandler.UpdateUser,
			)
		}

		// Workspace routes
		workspaces := v1.Group("/workspaces")
		{
			workspaces.POST("",
				auth.JWTMiddleware(),
				validation.ValidationMiddleware(validate, dto.WorkspaceCreateDTO{}),
				handlers.WorkspaceHandler.Create,
			)
			workspaces.GET("/:id",
				auth.JWTMiddleware(),
				handlers.WorkspaceHandler.GetByID,
			)
			workspaces.PATCH("/:id",
				auth.JWTMiddleware(),
				validation.ValidationMiddleware(validate, dto.WorkspaceUpdateDTO{}),
				handlers.WorkspaceHandler.Update,
			)
			workspaces.DELETE("/:id",
				auth.JWTMiddleware(),
				handlers.WorkspaceHandler.Delete,
			)
			workspaces.GET("",
				auth.JWTMiddleware(),
				handlers.WorkspaceHandler.List,
			)
		}

		// Thread routes
		threads := v1.Group("/threads")
		{
			threads.POST("",
				auth.JWTMiddleware(),
				validation.ValidationMiddleware(validate, dto.ThreadCreateDTO{}),
				handlers.ThreadHandler.Create,
			)
			threads.GET("/:id",
				auth.JWTMiddleware(),
				handlers.ThreadHandler.GetByID,
			)
			threads.GET("/workspace/:workspace_id",
				auth.JWTMiddleware(),
				handlers.ThreadHandler.GetThreadsByWorkspace,
			)
			threads.PATCH("/:id",
				auth.JWTMiddleware(),
				validation.ValidationMiddleware(validate, dto.ThreadUpdateDTO{}),
				handlers.ThreadHandler.Update,
			)
			threads.POST("/:id/messages",
				auth.JWTMiddleware(),
				validation.ValidationMiddleware(validate, dto.MessageDTO{}),
				handlers.ThreadHandler.AddMessage,
			)
			threads.DELETE("/:id",
				auth.JWTMiddleware(),
				handlers.ThreadHandler.Delete,
			)
		}
		return router, nil
	}
}
