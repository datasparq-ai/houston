package database

import (
	"context"
	"fmt"
	"github.com/datasparq-ai/houston/model"
	"github.com/go-redis/redis/v8"
	"strings"
)

type RedisDatabase struct {
	Database
	client *redis.Client
	ctx    context.Context
}

// NewRedisDatabase initialises a redis client using the default settings from ../config.go
func NewRedisDatabase(addr, password string, db int) *RedisDatabase {

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // default is no password
		DB:       db,       // default DB is 0
	})

	return &RedisDatabase{
		client: rdb,
		ctx:    context.Background(),
	}
}

// CreateKey does nothing when using Redis db
func (d *RedisDatabase) CreateKey(key string) error {
	return nil
}

// DeleteKey completely removes an API key and all associated plans and missions from the database
func (d *RedisDatabase) DeleteKey(key string) error {
	allKeys, err := d.client.Keys(d.ctx, key+"|*").Result()
	if err != nil {
		return err
	}

	for _, k := range allKeys {
		err = d.client.Del(d.ctx, k).Err()
	}
	return err
}

func (d *RedisDatabase) Set(key string, field string, value string) error {
	if field != "" {
		key += "|" + field
	}
	err := d.client.Set(d.ctx, key, value, 0).Err()
	if err == redis.Nil {
		return fmt.Errorf("key '%v' not found", key)
	} else {
		return err
	}
}

func (d *RedisDatabase) Get(key string, field string) (string, bool) {
	if field != "" {
		key += "|" + field
	}
	value, err := d.client.Get(d.ctx, key).Result()
	if err == redis.Nil {
		return "", false
	} else if err != nil {
		return value, false
	}
	return value, true
}

func (d *RedisDatabase) Delete(key string, field string) bool {
	if field != "" {
		key += "|" + field
	}
	_, err := d.client.Del(d.ctx, key, field).Result()
	// result of 1 means one field was deleted
	if err != nil {
		fmt.Printf("failed to delete %v: %v\n", key, field)
		return false
	}
	return true
}

func (d *RedisDatabase) Ping() error {
	err := d.client.Ping(d.ctx).Err()
	return err
}

func (d *RedisDatabase) ListKeys() ([]string, error) {
	value, err := d.client.Keys(d.ctx, "*|u").Result()
	if err != nil {
		return value, err
	}
	for i, s := range value {
		pipeIndex := strings.Index(s, "|")
		if pipeIndex > -1 {
			value[i] = s[:pipeIndex]
		}
	}
	return value, nil
}

func (d *RedisDatabase) List(key string, prefix string) ([]string, error) {
	value, err := d.client.Keys(d.ctx, key+"|"+prefix+"*").Result()
	if err != nil {
		return value, err
	}
	prefixLen := len(key) + 1
	for i, s := range value {
		value[i] = s[prefixLen:]
	}
	return value, nil
}

// DoTransaction watches a key, gets its value, performs an operation, and sets it again
func (d *RedisDatabase) DoTransaction(transactionFunc func(string) (string, error), key string, field string) error {

	err := d.client.Watch(d.ctx, func(tx *redis.Tx) error {

		value, err := tx.Get(d.ctx, key+"|"+field).Result()
		if err != nil && err != redis.Nil {
			return &model.KeyNotFoundError{}
		}

		value, err = transactionFunc(value)
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(d.ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(d.ctx, key+"|"+field, value, 0)
			return nil
		})
		return err
	}, key+"|"+field)

	if err == redis.TxFailedErr {
		return &model.TransactionFailedError{}
	}

	return err
}
