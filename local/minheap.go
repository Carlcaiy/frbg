package local

import (
	"errors"
)

type IMinHeap interface {
	Val() int64
	SetIndex(int)
	Index() int
}

type MinHeap struct {
	data []IMinHeap
}

func NewMinHeap() *MinHeap {
	return &MinHeap{
		data: make([]IMinHeap, 0, 8),
	}
}

func (m *MinHeap) Len() int {
	return len(m.data)
}

func (m *MinHeap) Top() IMinHeap {
	if len(m.data) == 0 {
		return nil
	}
	return m.data[0]
}

func (m *MinHeap) Push(e IMinHeap) error {
	m.data = append(m.data, e)
	m.shiftUp(len(m.data) - 1)

	for i := 0; i < len(m.data); i++ {
		m.data[i].SetIndex(i)
	}

	return nil
}

func (m *MinHeap) Pop() (d interface{}, err error) {
	if len(m.data) <= 0 {
		return nil, errors.New("heap empty")
	}
	tail := len(m.data) - 1
	d = m.data[0]
	m.data[0], m.data[tail] = m.data[tail], m.data[0]
	m.data = m.data[:tail]

	m.shiftDown(0)

	for i := 0; i < tail; i++ {
		m.data[i].SetIndex(i)
	}
	return d, nil
}

func (m *MinHeap) Drop(i IMinHeap) error {
	idx := i.Index()
	if len(m.data) <= idx {
		return errors.New("Drop heap error")
	}
	tail := len(m.data) - 1
	m.data[idx], m.data[tail] = m.data[tail], m.data[idx]
	m.data = m.data[:tail]

	m.shiftDown(idx)

	for i := idx; i < tail; i++ {
		m.data[i].SetIndex(i)
	}
	return nil
}

func (m *MinHeap) Build() {
	for i := len(m.data) / 2; i >= 0; i-- {
		m.shiftDown(0)
	}
	for i := range m.data {
		m.data[i].SetIndex(i)
	}
}

func (m *MinHeap) shiftDown(idx int) {
	length := len(m.data)
	for idx*2+1 < length {
		temp := idx*2 + 1
		if idx*2+2 < length {
			if m.data[idx*2+1].Val() > m.data[idx*2+2].Val() {
				temp = idx*2 + 2
			} else {
				temp = idx*2 + 1
			}
		}
		if m.data[idx].Val() > m.data[temp].Val() {
			m.data[idx], m.data[temp] = m.data[temp], m.data[idx]
			idx = temp
		} else {
			break
		}
	}
}

func (m *MinHeap) shiftUp(idx int) {
	for idx > 0 {
		root := (idx - 1) / 2
		if m.data[idx].Val() < m.data[root].Val() {
			m.data[idx], m.data[root] = m.data[root], m.data[idx]
			if idx%2 == 0 && m.data[idx-1].Val() < m.data[root].Val() {
				m.data[idx-1], m.data[root] = m.data[root], m.data[idx-1]
			}
		} else {
			break
		}
		idx = root
	}
}
