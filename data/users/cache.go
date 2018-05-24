package users

import (
	"context"
	"fmt"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/windmilleng/mish/data"
)

type userReaderCache struct {
	delegate data.UserReader
	cache    *cache.Cache
}

// Caches users for `ttl`, so we're not constantly making RPCs for
// the same user object.
func NewUserReaderCache(delegate data.UserReader, ttl time.Duration) userReaderCache {
	return userReaderCache{
		delegate: delegate,
		cache:    cache.New(ttl, ttl),
	}
}

func (c userReaderCache) LookupByUserID(ctx context.Context, userID data.UserID) (data.User, error) {
	key := fmt.Sprintf("%d", userID)
	obj, ok := c.cache.Get(key)
	if ok {
		user, ok := obj.(data.User)
		if ok {
			return user, nil
		}
	}

	user, err := c.delegate.LookupByUserID(ctx, userID)
	if err != nil {
		return data.User{}, err
	}

	c.cache.SetDefault(key, user)
	return user, err
}

var _ data.UserReader = userReaderCache{}
