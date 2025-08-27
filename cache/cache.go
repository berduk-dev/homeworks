package cache

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"time"
)

type LinksCache struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *LinksCache {
	return &LinksCache{
		rdb: rdb,
	}
}

func (c *LinksCache) StoreLink(shortLink string, longLink string) error {
	cmd := c.rdb.Set(shortLink, longLink, time.Hour)
	if cmd.Err() != nil {
		return fmt.Errorf("error StoreLink: %w", cmd.Err())
	}

	return nil
}

func (c *LinksCache) GetLink(shortLink string) (string, error) {
	cmd := c.rdb.Get(shortLink)
	if cmd.Err() != nil {
		if errors.Is(cmd.Err(), redis.Nil) {
			return "", nil
		}
		return "", fmt.Errorf("error GetLink: %w", cmd.Err())
	}

	return cmd.Val(), nil
}
