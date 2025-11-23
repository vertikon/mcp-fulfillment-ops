package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisClient encapsula o cliente Redis para cache e locks
type RedisClient struct {
	client *redis.Client
	logger Logger
}

// Logger define o contrato para logging
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

// NewRedisClient cria uma nova instância do cliente Redis
func NewRedisClient(url string) (*RedisClient, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Verificar conexão
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return &RedisClient{
		client: client,
	}, nil
}

// Get obtém um valor do cache
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key not found: %s", key)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get key: %w", err)
	}
	return val, nil
}

// Set define um valor no cache com TTL
func (r *RedisClient) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Delete remove uma chave do cache
func (r *RedisClient) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Lock tenta adquirir um lock distribuído
func (r *RedisClient) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	// Implementação simples de lock usando SET NX
	result, err := r.client.SetNX(ctx, key, "locked", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}
	return result, nil
}

// Unlock libera um lock distribuído
func (r *RedisClient) Unlock(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Close fecha a conexão com Redis
func (r *RedisClient) Close() error {
	return r.client.Close()
}
