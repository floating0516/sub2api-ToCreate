package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type installCredentialStore struct {
	rdb *redis.Client
}

var installCredentialTransitionScript = redis.NewScript(`
local status = redis.call("HGET", KEYS[1], "status")
if not status then
  return "missing"
end
if status ~= "pending" then
  return status
end
local expires_at = tonumber(redis.call("HGET", KEYS[1], "expires_at") or "0")
local now = tonumber(ARGV[1])
if expires_at <= now then
  redis.call("HSET", KEYS[1], "status", "expired")
  return "expired"
end
redis.call("HSET", KEYS[1], "status", ARGV[2])
if ARGV[2] == "redeemed" then
  redis.call("HSET", KEYS[1], "used_at", ARGV[1])
end
return "transitioned"
`)

var installCredentialRateLimitScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return current
`)

func NewInstallCredentialStore(rdb *redis.Client) service.InstallCredentialStore {
	return &installCredentialStore{rdb: rdb}
}

func (s *installCredentialStore) Save(
	ctx context.Context,
	storageKey string,
	record *service.InstallCredentialRecord,
	ttl time.Duration,
) error {
	if s.rdb == nil {
		return fmt.Errorf("redis client is not configured")
	}
	if record == nil {
		return fmt.Errorf("install credential record is nil")
	}
	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, storageKey, map[string]any{
		"kind":       record.Kind,
		"status":     record.Status,
		"user_id":    record.UserID,
		"key_id":     record.KeyID,
		"client":     record.Client,
		"created_at": record.CreatedAt.Unix(),
		"expires_at": record.ExpiresAt.Unix(),
	})
	pipe.Expire(ctx, storageKey, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *installCredentialStore) Load(
	ctx context.Context,
	storageKey string,
) (*service.InstallCredentialRecord, error) {
	if s.rdb == nil {
		return nil, fmt.Errorf("redis client is not configured")
	}
	values, err := s.rdb.HGetAll(ctx, storageKey).Result()
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil
	}

	userID, err := parseInstallCredentialInt64(values, "user_id")
	if err != nil {
		return nil, err
	}
	keyID, err := parseInstallCredentialInt64(values, "key_id")
	if err != nil {
		return nil, err
	}
	createdAt, err := parseInstallCredentialInt64(values, "created_at")
	if err != nil {
		return nil, err
	}
	expiresAt, err := parseInstallCredentialInt64(values, "expires_at")
	if err != nil {
		return nil, err
	}

	record := &service.InstallCredentialRecord{
		Kind:      values["kind"],
		Status:    values["status"],
		UserID:    userID,
		KeyID:     keyID,
		Client:    values["client"],
		CreatedAt: time.Unix(createdAt, 0).UTC(),
		ExpiresAt: time.Unix(expiresAt, 0).UTC(),
	}
	if usedAtRaw := values["used_at"]; usedAtRaw != "" {
		usedAt, parseErr := strconv.ParseInt(usedAtRaw, 10, 64)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid install credential used_at: %w", parseErr)
		}
		usedAtTime := time.Unix(usedAt, 0).UTC()
		record.UsedAt = &usedAtTime
	}
	return record, nil
}

func (s *installCredentialStore) Transition(
	ctx context.Context,
	storageKey string,
	now time.Time,
	nextStatus string,
) (string, error) {
	if s.rdb == nil {
		return "", fmt.Errorf("redis client is not configured")
	}
	result, err := installCredentialTransitionScript.Run(
		ctx,
		s.rdb,
		[]string{storageKey},
		now.UTC().Unix(),
		nextStatus,
	).Result()
	if err != nil {
		return "", err
	}
	return fmt.Sprint(result), nil
}

func (s *installCredentialStore) Delete(ctx context.Context, storageKey string) error {
	if s.rdb == nil {
		return fmt.Errorf("redis client is not configured")
	}
	return s.rdb.Del(ctx, storageKey).Err()
}

func (s *installCredentialStore) Increment(
	ctx context.Context,
	storageKey string,
	window time.Duration,
) (int64, error) {
	if s.rdb == nil {
		return 0, fmt.Errorf("redis client is not configured")
	}
	return installCredentialRateLimitScript.Run(
		ctx,
		s.rdb,
		[]string{storageKey},
		window.Milliseconds(),
	).Int64()
}

func parseInstallCredentialInt64(values map[string]string, key string) (int64, error) {
	value, err := strconv.ParseInt(values[key], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid install credential %s: %w", key, err)
	}
	return value, nil
}
