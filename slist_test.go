package gorqs

import (
	"testing"
)

type fakeJobber struct{}

func (f *fakeJobber) Run() error {
	return nil
}

func (f *fakeJobber) getID() int64 {
	return -1
}

func TestSyncList_IsEmpty(t *testing.T) {
	t.Run("check empty slist", func(t *testing.T) {
		l := list()
		yes := l.isEmpty()
		if !yes {
			t.Error("initialized slist should be empty")
		}
	})

	t.Run("check non-empty slist", func(t *testing.T) {
		l := list()
		l.push(&fakeJobber{})
		yes := l.isEmpty()
		if yes {
			t.Error("slist contains one item. should not be empty")
		}
	})
}

func TestSyncList_Push(t *testing.T) {
	l := list()
	count := 5
	for c := 0; c < count; c++ {
		l.push(&fakeJobber{})
	}
	got := l.count.Load()
	if got != int64(count) {
		t.Errorf("pushed %d jobs but slist contains %d jobs", count, got)
	}
}

func TestSyncList_Pop(t *testing.T) {
	t.Run("pop empty slist", func(t *testing.T) {
		l := list()
		got := l.pop()
		if got != nil {
			t.Errorf("pop initialized slist should return <nil>, but got %v", got)
			return
		}

		if count := l.count.Load(); count != 0 {
			t.Errorf("pop initialized slist have 0 item, but got %d", count)
		}
	})

	t.Run("pop non-empty slist", func(t *testing.T) {
		l := list()
		jobA := &fakeJobber{}
		jobB := &fakeJobber{}
		l.push(jobA)
		l.push(jobB)

		got := l.pop()
		if got != jobB {
			t.Errorf("got %v but expected %v", got, jobA)
			return
		}

		got = l.pop()
		if got != jobA {
			t.Errorf("got %v but expected %v", got, jobB)
			return
		}

		if count := l.count.Load(); count != 0 {
			t.Errorf("slist all items of slist. expect 0 remaining item, but got %d", count)
		}
	})
}
