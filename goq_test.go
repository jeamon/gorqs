package goq

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
	return New(MODE_SYNC, 3, 0)
}

func initializeAsyncQueue() Queuer {
	return New(MODE_ASYNC, 0, 0)
}

func check(t *testing.T, expect int, id int64, err error) {
	t.Helper()
	assert.NoError(t, err)
	assert.Equal(t, int64(expect), id)
}
