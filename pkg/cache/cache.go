package cache

import (
	"context"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
	"time"
)

func NewClient() *redis.Client {
	log.Print("Creating redis client")

	redisURI := os.Getenv("REDIS_URI")

	if redisURI == "" {
		log.Fatal("REDIS_URI is not set")
	}

	opts, err := redis.ParseURL(redisURI)
	if err != nil {
		log.Fatalf("Could not parse redis url due to error: %s", err.Error())
	}

	redisClient := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Could not connect to redis due to error: %s", err.Error())
	}

	log.Print("Created redis client")

	return redisClient
}
