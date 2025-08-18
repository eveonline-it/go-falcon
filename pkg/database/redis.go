package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"go-falcon/pkg/config"

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

	redis := &Redis{
		Client: client,
	}

	// Only initialize tracer if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		redis.tracer = otel.Tracer("redis-client")
	}

	return redis, nil
}

func (r *Redis) Close() error {
	return r.Client.Close()
}

func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if r.tracer != nil {
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

	return r.Client.Set(ctx, key, value, expiration).Err()
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	if r.tracer != nil {
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

	return r.Client.Get(ctx, key).Result()
}

func (r *Redis) Delete(ctx context.Context, keys ...string) error {
	if r.tracer != nil {
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

	return r.Client.Del(ctx, keys...).Err()
}

func (r *Redis) Exists(ctx context.Context, keys ...string) (int64, error) {
	if r.tracer != nil {
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

	return r.Client.Exists(ctx, keys...).Result()
}

func (r *Redis) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return r.Client.Ping(ctx).Err()
}

// SetJSON stores a JSON-serializable object in Redis with expiration
func (r *Redis) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if r.tracer != nil {
		ctx, span := r.tracer.Start(ctx, "redis.set_json",
			trace.WithAttributes(
				attribute.String("redis.key", key),
				attribute.String("redis.operation", "SET_JSON"),
				attribute.Int("redis.data_size", len(jsonData)),
			),
		)
		defer span.End()

		err := r.Client.Set(ctx, key, jsonData, expiration).Err()
		if err != nil {
			span.RecordError(err)
		}
		return err
	}

	return r.Client.Set(ctx, key, jsonData, expiration).Err()
}

// GetJSON retrieves and unmarshals a JSON object from Redis
func (r *Redis) GetJSON(ctx context.Context, key string, dest interface{}) error {
	var jsonData string
	var err error

	if r.tracer != nil {
		ctx, span := r.tracer.Start(ctx, "redis.get_json",
			trace.WithAttributes(
				attribute.String("redis.key", key),
				attribute.String("redis.operation", "GET_JSON"),
			),
		)
		defer span.End()

		jsonData, err = r.Client.Get(ctx, key).Result()
		if err != nil {
			span.RecordError(err)
			return err
		}
	} else {
		jsonData, err = r.Client.Get(ctx, key).Result()
		if err != nil {
			return err
		}
	}

	err = json.Unmarshal([]byte(jsonData), dest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// SetWithTTL sets a key with a TTL and returns the remaining TTL
func (r *Redis) SetWithTTL(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if r.tracer != nil {
		ctx, span := r.tracer.Start(ctx, "redis.set_with_ttl",
			trace.WithAttributes(
				attribute.String("redis.key", key),
				attribute.String("redis.operation", "SET_WITH_TTL"),
				attribute.String("redis.expiration", expiration.String()),
			),
		)
		defer span.End()

		err := r.Client.Set(ctx, key, value, expiration).Err()
		if err != nil {
			span.RecordError(err)
		}
		return err
	}

	return r.Client.Set(ctx, key, value, expiration).Err()
}

// GetTTL returns the remaining time to live for a key
func (r *Redis) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	if r.tracer != nil {
		ctx, span := r.tracer.Start(ctx, "redis.get_ttl",
			trace.WithAttributes(
				attribute.String("redis.key", key),
				attribute.String("redis.operation", "TTL"),
			),
		)
		defer span.End()

		ttl, err := r.Client.TTL(ctx, key).Result()
		if err != nil {
			span.RecordError(err)
		}
		return ttl, err
	}

	return r.Client.TTL(ctx, key).Result()
}