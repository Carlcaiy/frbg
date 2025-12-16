package util

import "errors"

// ArrayQueue 基于数组实现的队列
type ArrayQueue struct {
	capacity int           // 队列容量（数组大小，实际存储元素数为 capacity-1）
	array    []interface{} // 存储元素的数组
	front    int           // 队头指针（指向队头元素）
	rear     int           // 队尾指针（指向队尾元素的下一个位置）
}

// NewArrayQueue 创建指定容量的队列
func NewArrayQueue(capacity int) *ArrayQueue {
	if capacity < 2 { // 至少预留一个空位，容量需≥2
		capacity = 2
	}
	return &ArrayQueue{
		capacity: capacity,
		array:    make([]interface{}, capacity),
		front:    0,
		rear:     0,
	}
}

// Enqueue 入队操作
func (q *ArrayQueue) Enqueue(item interface{}) error {
	// 判断队列是否已满
	if (q.rear+1)%q.capacity == q.front {
		return errors.New("队列已满")
	}
	q.array[q.rear] = item             // 元素存入队尾
	q.rear = (q.rear + 1) % q.capacity // 队尾指针后移（循环）
	return nil
}

// Dequeue 出队操作
func (q *ArrayQueue) Dequeue() (interface{}, error) {
	// 判断队列是否为空
	if q.front == q.rear {
		return nil, errors.New("队列为空")
	}
	item := q.array[q.front]             // 获取队头元素
	q.array[q.front] = nil               // 清空原位置（可选，避免内存泄漏）
	q.front = (q.front + 1) % q.capacity // 队头指针后移（循环）
	return item, nil
}

// Front 获取队头元素（不出队）
func (q *ArrayQueue) Front() (interface{}, error) {
	if q.front == q.rear {
		return nil, errors.New("队列为空")
	}
	return q.array[q.front], nil
}

// IsEmpty 判断队列是否为空
func (q *ArrayQueue) IsEmpty() bool {
	return q.front == q.rear
}

// IsFull 判断队列是否已满
func (q *ArrayQueue) IsFull() bool {
	return (q.rear+1)%q.capacity == q.front
}

// Size 获取队列当前元素个数
func (q *ArrayQueue) Size() int {
	return (q.rear - q.front + q.capacity) % q.capacity
}
