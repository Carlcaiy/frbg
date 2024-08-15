package local

import (
	"log"
	"testing"
)

type XXX struct {
	val   int
	index int
}

func (x XXX) Val() int64 {
	return int64(x.val)
}

func (x *XXX) SetIndex(i int) {
	x.index = i
}

func (x XXX) Index() int {
	return x.index
}

func NewXXX(v int) *XXX {
	return &XXX{
		val: v,
	}
}

func TestHeap(t *testing.T) {
	h := NewMinHeap(50)
	h.Push(NewXXX(5))
	log.Println(h.data[:h.len])
	h.Push(NewXXX(8))
	log.Println(h.data[:h.len])
	h.Push(NewXXX(6))
	log.Println(h.data[:h.len])
	h.Push(NewXXX(3))
	log.Println(h.data[:h.len])
	h.Push(NewXXX(10))
	log.Println(h.data[:h.len])
	h.Push(NewXXX(1))
	log.Println(h.data[:h.len])
	h.Push(NewXXX(2))
	log.Println(h.data[:h.len])
	a, _ := h.Pop()
	log.Println(a, h.data[:h.len])
	a, _ = h.Pop()
	log.Println(a, h.data[:h.len])
	a, _ = h.Pop()
	log.Println(a, h.data[:h.len])
	a, _ = h.Pop()
	log.Println(a, h.data[:h.len])
	a, _ = h.Pop()
	log.Println(a, h.data[:h.len])
	a, _ = h.Pop()
	log.Println(a, h.data[:h.len])
	a, _ = h.Pop()
	log.Println(a, h.data[:h.len])
	a, _ = h.Pop()
	log.Println(a, h.data[:h.len])
}
