package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const PromptCacheTTL = 1 * time.Hour   // how long to cache prompt responses

func GetCachedResponse(client *redis.Client, prompt string) (string, bool) {
	ctx := context.Background()
	val, err := client.Get(ctx, "prompt:"+prompt).Result()    // exact match caching for prompts, could do embedding vector search for more flexible/paraphrased prompts as well 
	if err == redis.Nil {
		return "", false
	}
	if err != nil {
		return "", false
	}
	return val, true
}

func SetCachedResponse(client *redis.Client, prompt string, response string){
	ctx := context.Background()
	client.Set(ctx, "prompt:"+prompt, response, PromptCacheTTL)  // sets the response w/ the prompt as the key, exact match 
}
