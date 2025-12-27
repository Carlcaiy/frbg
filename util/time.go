package util

import (
	"sync/atomic"
	"time"
)

// 缓存的系统时间（用atomic保证并发安全，无需锁）
var cachedTime atomic.Value

// 初始化时间缓存，并定期更新
func init() {
	// 初始化缓存
	cachedTime.Store(time.Now())
	// 启动后台协程，每10ms更新一次（更新频率可根据业务精度要求调整）
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			cachedTime.Store(time.Now())
		}
	}()
}

// 获取缓存的时间（高性能，并发安全）
func Now() time.Time {
	return cachedTime.Load().(time.Time)
}
