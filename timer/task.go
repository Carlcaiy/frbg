package timer

import (
	"time"
)

// 最小堆事件事件

type Task struct {
	Loop        bool          // true循环任务 false延时任务
	duration    time.Duration // 时间间隔，延时时间
	triggerTime time.Time     // 触发时间
	event       func()        // 触发时间
	index       int           // 下标
}

func (t *Task) Val() int64 {
	return t.triggerTime.UnixNano()
}

func (t *Task) Index() int {
	return t.index
}

func (t *Task) SetIndex(i int) {
	t.index = i
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
