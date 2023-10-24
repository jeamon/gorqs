package gorqs

import (
	"sync"
	"sync/atomic"
)

// node is an element of a synchronized singly linked list (slist).
type node struct {
	job  jobber
	next *node
}

// slist represents a synchronized (concurent safe) singly linked-list.
type slist struct {
	mu    *sync.Mutex
	head  *node
	count atomic.Int64
}

// list provides a synchronized singly linked list to be used as internal job queue.
func list() *slist {
	return &slist{
		mu:   &sync.Mutex{},
		head: nil,
	}
}

// isEmpty tells
func (sl *slist) isEmpty() bool {
	return sl.count.Load() == 0
}

// push adds a job to the queue.
func (sl *slist) push(j jobber) {
	sl.mu.Lock()
	node := &node{
		job:  j,
		next: nil,
	}

	if sl.head == nil {
		sl.head = node
		sl.count.Add(1)
		sl.mu.Unlock()
		return
	}

	n := sl.head
	for n.next != nil {
		n = n.next
	}
	n.next = node
	sl.count.Add(1)
	sl.mu.Unlock()
}

// pop returns the oldest job from the queue.
func (sl *slist) pop() jobber {
	sl.mu.Lock()
	if sl.head == nil {
		sl.mu.Unlock()
		return nil
	}
	head := sl.head
	sl.head = sl.head.next
	sl.count.Add(-1)
	sl.mu.Unlock()
	return head.job
}
