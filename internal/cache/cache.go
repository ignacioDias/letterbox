package cache

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	glide "github.com/valkey-io/valkey-glide/go/v2"
	"github.com/valkey-io/valkey-glide/go/v2/config"
	"github.com/valkey-io/valkey-glide/go/v2/options"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, expiringTimeInSeconds int) error
	Delete(ctx context.Context, keys []string) (int64, error)
	CloseClient()
}

type cache struct {
	client *glide.Client
}

var ErrKeyNotFound = errors.New("key not found")

func NewCache() (Cache, error) {
	host := os.Getenv("VALKEY_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("VALKEY_PORT")
	if port == "" {
		port = "6379"
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("invalid VALKEY_PORT %q: %w", port, err)
	}

	useTLS := false
	if value := os.Getenv("VALKEY_USE_TLS"); value != "" {
		useTLS, err = strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid VALKEY_USE_TLS %q: %w", value, err)
		}
	}

	databaseID := 0
	if value := os.Getenv("VALKEY_DATABASE_ID"); value != "" {
		databaseID, err = strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid VALKEY_DATABASE_ID %q: %w", value, err)
		}
	}

	cfg := config.NewClientConfiguration().
		WithAddress(&config.NodeAddress{Host: host, Port: portNum}).
		WithRequestTimeout(5 * time.Second).
		WithUseTLS(useTLS).
		WithDatabaseId(databaseID)
	client, err := glide.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Valkey Glide client: %w", err)
	}
	return &cache{
		client: client,
	}, nil
}

func (c *cache) Get(ctx context.Context, key string) (string, error) {
	result, err := c.client.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if result.IsNil() {
		return "", ErrKeyNotFound
	}
	return result.Value(), nil
}

func (c *cache) Set(ctx context.Context, key string, value string, expiringTimeInSeconds int) error {
	if expiringTimeInSeconds <= 0 {
		return fmt.Errorf("expiring time makes no sense, must be greater than 0 seconds")
	}
	setOptions := options.SetOptions{
		Expiry: options.NewExpiryIn(time.Duration(expiringTimeInSeconds) * time.Second),
	}
	_, err := c.client.SetWithOptions(ctx, key, value, setOptions)
	return err
}

func (c *cache) Delete(ctx context.Context, keys []string) (int64, error) {
	return c.client.Del(ctx, keys)
}
func (c *cache) CloseClient() {
	if c.client != nil {
		c.client.Close()
	}
}
