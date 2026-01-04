package route

import "log"

const full = 0xf

const (
	TagFaPai uint32 = 1
	TagPlay  uint32 = 2
)

type Wait struct {
	tag  uint32 // 当前等待的标签
	flag int32
}

func NewWait(n int32) *Wait {
	return &Wait{}
}

func (w *Wait) Done(tag uint32, seat int32) bool {
	if w.tag != tag {
		log.Printf("wait tag not match, tag:%d, w.process:%d", tag, w.tag)
		return false
	}
	if w.flag&(1<<seat) != 0 {
		log.Printf("wait flag:%b, seat %d done, tag:%d, w.process:%d", w.flag, seat, tag, w.tag)
		return false
	}
	w.flag |= 1 << seat
	log.Printf("wait flag:%b, seat %d done, tag:%d, w.process:%d", w.flag, seat, tag, w.tag)
	return true
}

func (w *Wait) IsWait(seat int32) bool {
	return w.flag&(1<<seat) == 0
}

func (w *Wait) IsFull() bool {
	return w.flag == full
}

func (w *Wait) WaitAll(tag uint32) {
	w.tag = tag
	w.flag = 0
}

func (w *Wait) WaitOne(tag uint32, seat int32) {
	w.tag = tag
	w.flag = full ^ (1 << seat)
}

func (w *Wait) WaitOther(tag uint32, seat int32) {
	w.tag = tag
	w.flag |= 1 << seat
}

func (w *Wait) GetTag() uint32 {
	return w.tag
}

func (w *Wait) IsTag(tag uint32) bool {
	return w.tag == tag
}
