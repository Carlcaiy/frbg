package timer

import (
	"log"
	"time"
)

var timer *TaskCtl
var funcs []func()

// 初始化定时器
func Init(t time.Duration) {
	timer = NewTaskCtl(1024)
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

// 执行循环定时器
func StartLoopFunc(dur time.Duration, f func()) {
	timer.Push(&Task{
		duration: dur,
		Loop:     true,
		event:    f,
		index:    -1,
	})
}

// 执行延时定时器
func StartDelayFunc(dur time.Duration, f func()) {
	timer.Push(&Task{
		duration: dur,
		Loop:     false,
		event:    f,
		index:    -1,
	})
}

// 执行时间任务
func Start(e *Task) {
	timer.Push(e)
}

// 停止时间任务
func Stop(e *Task) {
	timer.Stop(e)
}

// 检测定时任务
func FrameCheck() {
	timer.FrameCheck()
}

// 添加定时任务
func NewTask(dur time.Duration, f func(), loop bool) *Task {
	return &Task{
		duration: dur,
		Loop:     loop,
		event:    f,
		index:    -1,
	}
}

// 添加定时任务
func NewLoopTask(dur time.Duration, f func()) *Task {
	return NewTask(dur, f, true)
}

// 添加定时任务
func NewDelayTask(dur time.Duration, f func()) *Task {
	return NewTask(dur, f, false)
}
