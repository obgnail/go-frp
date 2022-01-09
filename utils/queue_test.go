package utils

import (
	"testing"
	"time"
)

func TestQueue(t *testing.T) {
	q := NewQueue()
	go func() { q.Push("one") }()
	go func() { q.Push("four") }()
	q.Push("two")
	q.Push("three")
	t.Log((q.Pop()).(string) == "two")
	t.Log(q.Front().(string) == "three")
	t.Log((q.Pop()).(string) == "three")
	t.Log((q.Pop()).(string) == "one")
	t.Log((q.Pop()).(string) == "four")

	time.Sleep(2 * time.Second)
}
