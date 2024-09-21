package timer

import (
	"log"
	"time"
)

var funcs []func()

// 初始化定时器
func Init(t time.Duration) {
	go func() {
		log.Printf("start ticker(%v) coroutine", t)
		ticker := time.NewTicker(t)
		for range ticker.C {
			for _, f := range funcs {
				f()
			}
		}
	}()
}

// 添加事件触发源
func AddTrigger(f func()) {
	funcs = append(funcs, f)
}
