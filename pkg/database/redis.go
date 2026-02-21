package database

import (
	"coder_edu_backend/internal/config"
	"context"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
)

func InitRedis(cfg *config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     50,
		MinIdleConns: 5,
	})

	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	log.Println("Redis connection established")
	return rdb, nil
}
