package cache

import (
	"sync"
	"time"
)

type ExpiredCache[T any] struct {
	data      *T    // 数据
	expiredAt int64 // 过期时间戳，秒（unix）
	lock      sync.Mutex

	reloadFunc func() (*T, int64, error) // 重载参数
}

func NewExpiredCache[T any](reloadFunc func() (*T, int64, error)) (*ExpiredCache[T], error) {
	expiredCache := &ExpiredCache[T]{
		expiredAt:  -1,
		reloadFunc: reloadFunc,
		lock:       sync.Mutex{},
	}

	return expiredCache, nil
}

func (expiredCache *ExpiredCache[T]) Get() (*T, error) {
	if !expiredCache.expired() {
		return expiredCache.data, nil
	}

	// 加入锁，防止重复申请
	expiredCache.lock.Lock()
	defer func() {
		expiredCache.lock.Unlock()
	}()

	if !expiredCache.expired() {
		return expiredCache.data, nil
	}

	data, expiredAt, err := expiredCache.reloadFunc()
	if err != nil {
		return expiredCache.data, err
	}

	expiredCache.expiredAt = expiredAt
	expiredCache.data = data

	return data, nil
}

func (expiredCache *ExpiredCache[T]) expired() bool {
	return expiredCache.expiredAt < time.Now().Unix()
}
