package utils

import (
	"container/list"
	"sync"
)

var lock sync.Mutex

type Queue struct {
	data *list.List
}

func NewQueue() *Queue {
	q := new(Queue)
	q.data = list.New()
	return q
}

func (q *Queue) Peek() interface{} {
	defer lock.Unlock()
	lock.Lock()
	return q.data.Back().Value
}

func (q *Queue) Push(v interface{}) {
	defer lock.Unlock()
	lock.Lock()
	q.data.PushFront(v)
}

func (q *Queue) Pop() interface{} {
	defer lock.Unlock()
	lock.Lock()
	iter := q.data.Back()
	v := iter.Value
	q.data.Remove(iter)
	return v
}

func (q *Queue) Len() int {
	defer lock.Unlock()
	lock.Lock()
	return q.data.Len()
}
