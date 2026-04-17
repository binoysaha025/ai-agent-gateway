package cache

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

func Connect(redisURL string) *redis.Client {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Error parsing Redis URL: ", err)
	}

	client := redis.NewClient(opts)

	err = client.Ping(context.Background()).Err()
	if err != nil {
		log.Fatal("Error connecting to Redis: ", err)
	}

	log.Println("Connected to Redis")
	return client
}