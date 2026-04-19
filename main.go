package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/binoysaha025/ai-agent-gateway/agent"
	"github.com/binoysaha025/ai-agent-gateway/cache"
	"github.com/binoysaha025/ai-agent-gateway/config"
	"github.com/binoysaha025/ai-agent-gateway/db"
	"github.com/binoysaha025/ai-agent-gateway/handlers"
	"github.com/binoysaha025/ai-agent-gateway/middleware"
	"github.com/binoysaha025/ai-agent-gateway/models"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	cfg := config.Load()

	database := db.Connect(cfg.PostgresURL)
	defer database.Close()

	err = models.CreateAPIKeyTable(database)
	if err != nil {
		log.Fatal("Error creating API key table: ", err)
	}
	log.Println("tables ready")

	redisClient := cache.Connect(cfg.RedisURL)
	defer redisClient.Close()

	cb := middleware.NewCircuitBreaker(5, 30*time.Second) // circuit breaker with 5 failures and 30s timeout

	r := gin.Default()

	// unprotected routes
	h := handlers.NewHandler(database)
	r.POST("/keys", h.CreateAPIKey)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"port":   cfg.Port,
		})
	})
	
	r.POST("/embed", h.EmbedDocument)
	// protected routes
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(database))
	protected.Use(middleware.RateLimitMiddleware(redisClient))
	protected.Use(middleware.CircuitBreakerMiddleware(cb))
	{
		protected.POST("/query", func(c *gin.Context) {
			var body struct {
				Prompt string `json:"prompt"`
			}
			if err := c.ShouldBindJSON(&body); err != nil || body.Prompt == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "prompt required"})
				return
			}

			// check prompt cache
			if cached, hit := cache.GetCachedResponse(redisClient, body.Prompt); hit {
				c.JSON(200, gin.H{
					"response": cached,
					"cached":   true,
				})
				return
			}

			// run agent
			response, tokens, err := agent.Run(body.Prompt, database)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// cache the response
			cache.SetCachedResponse(redisClient, body.Prompt, response)

			c.JSON(200, gin.H{
				"response": response,
				"cached":   false,
				"tokens":   tokens,
			})
		})
	}

	log.Println("Starting server on port " + cfg.Port)
	r.Run(":" + cfg.Port)
}