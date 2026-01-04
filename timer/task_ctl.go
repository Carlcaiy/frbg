package timer

import (
	"frbg/util"
	"log"
	"time"
)

// 最小堆管理定时事件

type TaskCtl struct {
	*util.MinHeap
}

func NewTaskCtl() *TaskCtl {
	return &TaskCtl{
		MinHeap: util.NewMinHeap(),
	}
}

func (t *TaskCtl) Push(e *Task) {
	t.MinHeap.Push(e)
}

func (t *TaskCtl) Get() *Task {
	return t.MinHeap.Top().(*Task)
}

func (t *TaskCtl) Len() int {
	return t.MinHeap.Len()
}

func (t *TaskCtl) Pop() *Task {
	if data, err := t.MinHeap.Pop(); err != nil {
		log.Printf("Pop() error:%s", err.Error())
		return nil
	} else {
		return data.(*Task)
	}
}

func (t *TaskCtl) FrameCheck() {
	for t.Len() > 0 && t.Get().Val() <= time.Now().UnixNano() {
		if e := t.Pop(); e != nil {
			e.event()
			if e.Loop {
				e.triggerTime = e.triggerTime.Add(e.duration)
				t.Push(e)
			}
		}
	}
}

func (t *TaskCtl) Stop(e *Task) {
	t.Drop(e)
}

func (t *TaskCtl) Start(e *Task) {
	if e.index >= 0 {
		log.Println("timer hava started, use reset")
		return
	}
	e.triggerTime = time.Now().Add(e.duration)
	t.Push(e)
}

func (t *TaskCtl) Reset(e *Task) {
	t.Drop(e)
	e.triggerTime = time.Now().Add(e.duration)
	t.Push(e)
}
