package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Redis struct {
	Client *redis.Client
	tracer trace.Tracer
}

func NewRedis(ctx context.Context) (*Redis, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	log.Printf("Connected to Redis at: %s", opt.Addr)

	return &Redis{
		Client: client,
		tracer: otel.Tracer("redis-client"),
	}, nil
}

func (r *Redis) Close() error {
	return r.Client.Close()
}

func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	ctx, span := r.tracer.Start(ctx, "redis.set",
		trace.WithAttributes(
			attribute.String("redis.key", key),
			attribute.String("redis.operation", "SET"),
		),
	)
	defer span.End()

	err := r.Client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		span.RecordError(err)
	}
	return err
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	ctx, span := r.tracer.Start(ctx, "redis.get",
		trace.WithAttributes(
			attribute.String("redis.key", key),
			attribute.String("redis.operation", "GET"),
		),
	)
	defer span.End()

	result, err := r.Client.Get(ctx, key).Result()
	if err != nil {
		span.RecordError(err)
	}
	return result, err
}

func (r *Redis) Delete(ctx context.Context, keys ...string) error {
	ctx, span := r.tracer.Start(ctx, "redis.delete",
		trace.WithAttributes(
			attribute.StringSlice("redis.keys", keys),
			attribute.String("redis.operation", "DEL"),
		),
	)
	defer span.End()

	err := r.Client.Del(ctx, keys...).Err()
	if err != nil {
		span.RecordError(err)
	}
	return err
}

func (r *Redis) Exists(ctx context.Context, keys ...string) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "redis.exists",
		trace.WithAttributes(
			attribute.StringSlice("redis.keys", keys),
			attribute.String("redis.operation", "EXISTS"),
		),
	)
	defer span.End()

	result, err := r.Client.Exists(ctx, keys...).Result()
	if err != nil {
		span.RecordError(err)
	}
	return result, err
}

func (r *Redis) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return r.Client.Ping(ctx).Err()
}