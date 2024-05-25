package cache

import (
	"time"
)

func NewReloadCache[T any](reloadFunc func() (*T, error), useLastCache bool, timer time.Duration) (*ReloadCache[T], error) {
	reloadCache := &ReloadCache[T]{
		expired:      true,
		useLastCache: useLastCache,
		reloadFunc:   reloadFunc,
	}

	err := reloadCache.setTimer(timer)
	if err != nil {
		return nil, err
	}

	return reloadCache, nil
}

type ReloadCache[T any] struct {
	data         *T   // 数据
	expired      bool // 是否数据过期
	useLastCache bool // 加载失败后，是否使用上次缓存

	reloadFunc func() (*T, error) // 重载参数
}

func (reloadCache *ReloadCache[T]) Get() (*T, bool) {
	return reloadCache.data, !reloadCache.expired
}

func (reloadCache *ReloadCache[T]) setTimer(timer time.Duration) error {
	data, err := reloadCache.reloadFunc()
	if err == nil {
		reloadCache.data = data
		reloadCache.expired = false
	} else {
		return err
	}

	t := time.NewTicker(timer)

	go func() {
		for {
			<-t.C

			data, err = reloadCache.reloadFunc()
			if err != nil {
				if !reloadCache.useLastCache {
					reloadCache.expired = true
				}
			} else {
				reloadCache.data = data
				reloadCache.expired = false
			}
		}
	}()

	return nil
}
