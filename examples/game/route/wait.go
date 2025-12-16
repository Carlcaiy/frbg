package route

const full = 0xf

type Wait struct {
	tag  uint32
	flag int32
}

func NewWait(n int32) *Wait {
	return &Wait{}
}

func (w *Wait) Done(tag uint32, seat int32) bool {
	if w.tag != tag {
		return false
	}
	if w.flag&(1<<seat) != 0 {
		return false
	}
	w.flag |= 1 << seat
	return true
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
	w.flag |= 1 << seat
}
