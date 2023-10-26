package gorqs

import (
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Ensure concrete type Queue satisfies Queuer interface.
func TestQueuerInterface(t *testing.T) {
	var i interface{} = new(Queue)
	if _, ok := i.(Queuer); !ok {
		t.Fatalf("expected %T to implement Queuer", i)
	}
}

// Ensure concrete type job satisfies jobber interface.
func TestJobberInterface(t *testing.T) {
	var i interface{} = new(job)
	if _, ok := i.(jobber); !ok {
		t.Fatalf("expected %T to implement jobber", i)
	}
}

// Ensure calling Clear on queue empty results cache.
func TestQueue_Clear(t *testing.T) {
	q := &Queue{}
	q.records.Store(1, errors.New("job id 1 execution error"))
	q.Clear()
	if _, has := q.records.Load(1); has {
		t.Fatalf("expected to not find job id 1")
	}
}

func TestQueue_Result(t *testing.T) {
	id := int64(1)
	t.Run("no found", func(t *testing.T) {
		q := New(MODE_SYNC | TRACK_JOBS)
		err := q.Result(context.Background(), id)
		if err != ErrNotFound {
			t.Fatalf("expect ErrNotFound but got %v", err)
		}
	})

	t.Run("nil", func(t *testing.T) {
		q := New(MODE_SYNC | TRACK_JOBS)
		q.records.Store(id, nil)
		r := q.Result(context.Background(), id)
		if r != nil {
			t.Fatalf("expect <nil> but got %v", r)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		q := New(MODE_SYNC | TRACK_JOBS)
		q.records.Store(id, "no error type")
		err := q.Result(context.Background(), id)
		if err != ErrInvalid {
			t.Fatalf("expect ErrInvalid but got %v", err)
		}
	})

	t.Run("ok", func(t *testing.T) {
		err := errors.New("job execution error")
		qq := New(MODE_SYNC | TRACK_JOBS)
		qq.records.Store(id, err)
		result := qq.Result(context.Background(), id)
		if result != err {
			t.Fatalf("expect %v but got %v", err, result)
		}
	})
}

// Ensure all jobs added are queued and executed in the order they were added.
func TestSyncQueue_Basic(t *testing.T) {
	queue := initializeSyncQueue()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	done := make(chan struct{}, 1)
	go func() {
		err := queue.Start(ctx)
		assert.NoError(t, err)
		done <- struct{}{}
	}()

	results := make([]string, 0, 3)
	makeJob := func(id string) Runner {
		return basicTestJob(func() error {
			time.Sleep(time.Second)
			log.Printf("Finished -> %s", id)
			results = append(results, id)
			return nil
		})
	}

	waitQueue(t, queue)

	id, err := queue.Push(ctx, makeJob("job1"))
	check(t, 1, id, err)
	id, err = queue.Push(ctx, makeJob("job2"))
	check(t, 2, id, err)
	id, err = queue.Push(ctx, makeJob("job3"))
	check(t, 3, id, err)

	time.Sleep(time.Second * 4)

	err = queue.Stop(ctx)
	assert.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("Started queue did not exit.")
	}

	if !assert.Len(t, results, 3) {
		return
	}
	assert.Equal(t, "job1", results[0])
	assert.Equal(t, "job2", results[1])
	assert.Equal(t, "job3", results[2])
}

// Ensure all jobs added are queued and executed in the order they were added.
func TestSyncQueue_Result(t *testing.T) {
	queue := New(MODE_SYNC | TRACK_JOBS)
	ctx := context.Background()
	go func() {
		err := queue.Start(ctx)
		assert.NoError(t, err)
	}()

	makeJob := func(id string) Runner {
		return basicTestJob(func() error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})
	}

	waitQueue(t, queue)

	id, err := queue.Push(ctx, makeJob("job1"))
	check(t, 1, id, err)
	id, err = queue.Push(ctx, makeJob("job2"))
	check(t, 2, id, err)
	id, err = queue.Push(ctx, makeJob("job3"))
	check(t, 3, id, err)

	time.Sleep(100 * time.Millisecond)
	err = queue.Stop(ctx)
	assert.NoError(t, err)

	r := queue.Result(context.Background(), 1)
	assert.Nil(t, r)
	r = queue.Result(context.Background(), 2)
	assert.Equal(t, nil, r)
	r = queue.Result(context.Background(), 3)
	assert.Nil(t, r)
}

// Ensure all jobs pushed are immediately handled and executed concurrently.
func TestAsyncQueue_Basic(t *testing.T) {
	queue := initializeAsyncQueue()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	done := make(chan struct{}, 1)
	go func() {
		err := queue.Start(ctx)
		assert.NoError(t, err)
		done <- struct{}{}
	}()

	results := make([]string, 0, 3)
	makeJob := func(id string) Runner {
		return basicTestJob(func() error {
			time.Sleep(time.Second)
			log.Printf("Finished -> %s", id)
			results = append(results, id)
			return nil
		})
	}

	waitQueue(t, queue)
	id, err := queue.Push(ctx, makeJob("job1"))
	check(t, 1, id, err)
	id, err = queue.Push(ctx, makeJob("job2"))
	check(t, 2, id, err)
	id, err = queue.Push(ctx, makeJob("job3"))
	check(t, 3, id, err)

	time.Sleep(1500 * time.Millisecond)

	err = queue.Stop(ctx)
	assert.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("Started queue did not exit.")
	}

	if !assert.Len(t, results, 3) {
		return
	}
	assert.ElementsMatch(t, []string{"job1", "job2", "job3"}, results)
}

// Ensure pending job into the internal queue will not be executed once the Queue is stopped.
func TestSyncQueue_StopOngoing(t *testing.T) {
	queue := initializeSyncQueue()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	done := make(chan struct{}, 1)
	go func() {
		err := queue.Start(ctx)
		assert.NoError(t, err)
		done <- struct{}{}
	}()

	results := make([]string, 0, 2)
	makeJob := func(id string) Runner {
		return basicTestJob(func() error {
			time.Sleep(time.Second)
			log.Printf("Finished -> %s", id)
			results = append(results, id)
			return nil
		})
	}

	waitQueue(t, queue)
	id, err := queue.Push(ctx, makeJob("job1"))
	check(t, 1, id, err)
	id, err = queue.Push(ctx, makeJob("job2"))
	check(t, 2, id, err)
	id, err = queue.Push(ctx, makeJob("job3"))
	check(t, 3, id, err)

	time.Sleep(2500 * time.Millisecond)

	err = queue.Stop(ctx)
	assert.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("Started queue did not exit.")
	}

	if !assert.Len(t, results, 2) {
		return
	}
	assert.Equal(t, "job1", results[0])
	assert.Equal(t, "job2", results[1])
}

// Ensure pending job into the internal queue will not be executed once the Queue max uptime reached.
func TestSyncQueue_TimeoutOngoing(t *testing.T) {
	queue := initializeSyncQueue()
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	done := make(chan struct{}, 1)
	go func() {
		err := queue.Start(ctx)
		assert.Error(t, err)
		assert.EqualError(t, err, context.DeadlineExceeded.Error())
		done <- struct{}{}
	}()

	results := make([]string, 0, 2)
	makeJob := func(id string) Runner {
		return basicTestJob(func() error {
			time.Sleep(time.Second)
			log.Printf("Finished -> %s", id)
			results = append(results, id)
			return nil
		})
	}
	waitQueue(t, queue)
	id, err := queue.Push(ctx, makeJob("job1"))
	check(t, 1, id, err)
	id, err = queue.Push(ctx, makeJob("job2"))
	check(t, 2, id, err)
	id, err = queue.Push(ctx, makeJob("job3"))
	check(t, 3, id, err)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Error("Started queue did not exit.")
	}

	if !assert.Len(t, results, 2) {
		return
	}
	assert.Equal(t, "job1", results[0])
	assert.Equal(t, "job2", results[1])
}

type basicTestJob func() error

func (b basicTestJob) Run() error {
	return b()
}

func initializeSyncQueue() Queuer {
	return New(MODE_SYNC)
}

func initializeAsyncQueue() Queuer {
	return New(MODE_ASYNC)
}

func check(t *testing.T, expect int, id int64, err error) {
	t.Helper()
	assert.NoError(t, err)
	assert.Equal(t, int64(expect), id)
}

func waitQueue(t *testing.T, queue Queuer) {
	t.Helper()
	for !queue.IsRunning() {
		time.Sleep(1 * time.Microsecond)
	}
}
