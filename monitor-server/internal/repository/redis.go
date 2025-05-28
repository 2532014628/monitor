package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"monitor-server/internal/config"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// 定义错误
var (
	ErrKeyNotFound = fmt.Errorf("key not found")
)

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(cfg *config.Config) (*RedisRepository, error) {
	db, err := strconv.Atoi(cfg.Redis.DB)
	if err != nil {
		return nil, fmt.Errorf("转换Redis数据库失败: %v", err)
	}
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       db,
	})

	// 测试连接
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接Redis失败: %v", err)
	}

	return &RedisRepository{client: client}, nil
}

// SaveToken 保存token
func (r *RedisRepository) SaveToken(ctx context.Context, hostname, token string) error {
	key := fmt.Sprintf("agent:token:%s", hostname)
	return r.client.Set(ctx, key, token, 24*time.Hour).Err()
}

// GetToken 获取token
func (r *RedisRepository) GetToken(ctx context.Context, hostname string) (string, error) {
	key := fmt.Sprintf("agent:token:%s", hostname)
	return r.client.Get(ctx, key).Result()
}

// DeleteToken 删除token
func (r *RedisRepository) DeleteToken(ctx context.Context, hostname string) error {
	key := fmt.Sprintf("agent:token:%s", hostname)
	return r.client.Del(ctx, key).Err()
}

// SaveAgentStatus 保存agent状态
func (r *RedisRepository) SaveAgentStatus(ctx context.Context, hostname string, status bool) error {
	key := fmt.Sprintf("agent:status:%s", hostname)
	return r.client.Set(ctx, key, status, 5*time.Minute).Err()
}

// GetAgentStatus 获取agent状态
func (r *RedisRepository) GetAgentStatus(ctx context.Context, hostname string) (bool, error) {
	key := fmt.Sprintf("agent:status:%s", hostname)
	val, err := r.client.Get(ctx, key).Bool()
	if err == redis.Nil {
		return false, nil
	}
	return val, err
}

// ListAgents 列出所有agent
func (r *RedisRepository) ListAgents(ctx context.Context) ([]string, error) {
	pattern := "agent:token:*"
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	agents := make([]string, 0, len(keys))
	for _, key := range keys {
		hostname := key[len("agent:token:"):]
		agents = append(agents, hostname)
	}

	return agents, nil
}

// Scan 扫描键
func (r *RedisRepository) Scan(ctx context.Context, cursor uint64, pattern string, count int64) ([]string, uint64, error) {
	return r.client.Scan(ctx, cursor, pattern, count).Result()
}

// TTL 获取键的剩余生存时间
func (r *RedisRepository) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// Get 获取键的值
func (r *RedisRepository) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

// Delete 删除键
func (r *RedisRepository) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// AddSystemInfo 添加系统信息
func (r *RedisRepository) AddSystemInfo(ctx context.Context, request RequestData) error {
	timestamp := time.Now().Unix() // 获取当前时间戳
	key := fmt.Sprintf("system_info:%s:%d", request.HostInfo.Hostname, timestamp)
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal data to JSON: %s", err)
	}
	err = r.client.Set(ctx, key, jsonData, 30*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to insert data into Redis: %s", err)
	}
	return nil
}

// GetLastUpdateTime 获取主机的最后更新时间
func (r *RedisRepository) GetLastUpdateTime(ctx context.Context, key string) (string, error) {
	// 获取主机的最后更新时间
	lastUpdatedStr, err := r.client.HGet(ctx, key, "last_updated").Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrKeyNotFound
		}
		return "", fmt.Errorf("获取最后更新时间失败: %v", err)
	}

	return lastUpdatedStr, nil
}
