package cache

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	glide "github.com/valkey-io/valkey-glide/go/v2"
	"github.com/valkey-io/valkey-glide/go/v2/config"
	"github.com/valkey-io/valkey-glide/go/v2/options"
)

type Cache interface {
	Get(ctx context.Context, key string) (any, error)
	Set(ctx context.Context, key string, value string, expiringTimeInSeconds int) error
	Delete(ctx context.Context, keys []string) (int64, error)
	CloseClient()
}

type cache struct {
	client *glide.Client
}

func NewCache() Cache {
	host := os.Getenv("VALKEY_HOST")
	port := os.Getenv("VALKEY_PORT")
	portNum, err := strconv.Atoi(port)
	if err != nil {
		panic(fmt.Sprintf("Failed to convert port to integer: %v", err))
	}

	useTLSEnv := os.Getenv("VALKEY_USE_TLS")
	useTLS := useTLSEnv == "true"
	databaseIdEnv := os.Getenv("VALKEY_DATABASE_ID")
	databaseId, err := strconv.Atoi(databaseIdEnv)
	if err != nil {
		panic(fmt.Sprintf("Failed to convert database ID to integer: %v", err))
	}
	config := config.NewClientConfiguration().
		WithAddress(&config.NodeAddress{Host: host, Port: portNum}).
		WithRequestTimeout(5 * time.Second).
		WithUseTLS(useTLS).
		WithDatabaseId(databaseId)
	client, err := glide.NewClient(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Valkey Glide client: %v", err))
	}
	return &cache{
		client: client,
	}

}

func (c *cache) Get(ctx context.Context, key string) (any, error) {
	return c.client.Get(ctx, key)
}

func (c *cache) Set(ctx context.Context, key string, value string, expiringTimeInSeconds int) error {
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
