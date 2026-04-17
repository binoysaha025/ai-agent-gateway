package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/binoysaha025/ai-agent-gateway/models"
)

func RateLimitMiddleware(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get("api_key")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unauthenticated"})
			c.Abort()
			return
		}

		apiKey := val.(*models.APIKey)

		// Determine rate limit based on plan
		var limit int
		if apiKey.Plan == "pro" {
			limit = 100
		} else {
			limit = 10
		}

		// redis key for this api key's counter
		redisKey := fmt.Sprintf("rate:%s", apiKey.Key)

		ctx := context.Background()

		// increment counter 
		count, err := redisClient.Incr(ctx, redisKey).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check rate limit"})
			c.Abort()
			return
		}

		// set expiry of 1 min on first request
		if count == 1 {
			redisClient.Expire(ctx, redisKey, time.Minute)
		}

		// check if over limit
		if int(count) > limit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"limit": limit,
				"reset_in": "1 min",
			})
			c.Abort()
			return
		}

		// attach usage info to context
		c.Set("request_count", count)
		c.Set("rate_limit", limit)
		
		c.Next()
	}
}