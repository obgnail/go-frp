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

func (q *Queue) Front() interface{} {
	defer lock.Unlock()
	lock.Lock()
	return q.data.Front().Value
}

func (q *Queue) Push(v interface{}) {
	defer lock.Unlock()
	lock.Lock()
	q.data.PushBack(v)
}

func (q *Queue) Pop() interface{} {
	defer lock.Unlock()
	lock.Lock()
	iter := q.data.Front()
	v := iter.Value
	q.data.Remove(iter)
	return v
}

func (q *Queue) Len() int {
	defer lock.Unlock()
	lock.Lock()
	return q.data.Len()
}
