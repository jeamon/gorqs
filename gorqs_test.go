package gorqs

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
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

func TestQueue_Fetch(t *testing.T) {
	id := int64(1)
	t.Run("no found", func(t *testing.T) {
		q := New(SyncMode | TrackJobs)
		err := q.Fetch(context.Background(), id)
		if err != ErrNotFound {
			t.Fatalf("expect ErrNotFound but got %v", err)
		}
	})

	t.Run("nil", func(t *testing.T) {
		q := New(SyncMode | TrackJobs)
		q.records.Store(id, nil)
		r := q.Fetch(context.Background(), id)
		if r != nil {
			t.Fatalf("expect <nil> but got %v", r)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		q := New(SyncMode | TrackJobs)
		q.records.Store(id, "no error type")
		err := q.Fetch(context.Background(), id)
		if err != ErrInvalid {
			t.Fatalf("expect ErrInvalid but got %v", err)
		}
	})

	t.Run("ok", func(t *testing.T) {
		err := errors.New("job execution error")
		qq := New(SyncMode | TrackJobs)
		qq.records.Store(id, err)
		result := qq.Fetch(context.Background(), id)
		if result != err {
			t.Fatalf("expect %v but got %v", err, result)
		}
	})
}

// Ensure all jobs added are queued and executed in the order they were added.
func TestSyncQueue_Basic(t *testing.T) {
	queue := New(SyncMode)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{}, 1)
	go func() {
		if err := queue.Start(ctx); err != nil {
			t.Errorf("expected no error but got %v", err)
		}
		done <- struct{}{}
	}()

	results := make([]string, 0, 3)
	makeJob := func(id string) Runner {
		return basicTestJob(func() error {
			time.Sleep(10 * time.Millisecond)
			results = append(results, id)
			return nil
		})
	}

	waitUntilStarted(t, queue)

	id, err := queue.Push(ctx, makeJob("job1"))
	check(t, 1, id, err)
	id, err = queue.Push(ctx, makeJob("job2"))
	check(t, 2, id, err)
	id, err = queue.Push(ctx, makeJob("job3"))
	check(t, 3, id, err)

	// give more time for above josb to finish.
	time.Sleep(70 * time.Millisecond)

	if err := queue.Stop(ctx); err != nil {
		t.Errorf("expected <nil> but got %v", err)
	}

	select {
	case <-done:
	case <-time.After(20 * time.Millisecond):
		t.Error("running queue did not exit.")
	}

	if lg := len(results); lg != 3 {
		t.Fatalf("invalid results length. expected 3 but got %d", lg)
		return
	}

	if equal := reflect.DeepEqual([]string{"job1", "job2", "job3"}, results); !equal {
		t.Errorf("expected %v but got %v", []string{"job1", "job2", "job3"}, results)
	}
}

// Ensure a sync queue service provides exact executed jobs results.
func TestSyncQueue_Fetch(t *testing.T) {
	queue := New(SyncMode | TrackJobs)
	ctx := context.Background()
	go func() {
		err := queue.Start(ctx)
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}
	}()

	makeJob := func(id string) Runner {
		return basicTestJob(func() error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})
	}

	waitUntilStarted(t, queue)

	id, err := queue.Push(ctx, makeJob("job1"))
	check(t, 1, id, err)
	id, err = queue.Push(ctx, makeJob("job2"))
	check(t, 2, id, err)
	id, err = queue.Push(ctx, makeJob("job3"))
	check(t, 3, id, err)

	time.Sleep(100 * time.Millisecond)
	if err := queue.Stop(ctx); err != nil {
		t.Errorf("expected no error but got %v", err)
	}

	if r := queue.Fetch(context.Background(), 1); r != nil {
		t.Errorf("expected <nil> but got %v", r)
	}

	if r := queue.Fetch(context.Background(), 2); r != nil {
		t.Errorf("expected <nil> but got %v", r)
	}

	if r := queue.Fetch(context.Background(), 3); r != nil {
		t.Errorf("expected <nil> but got %v", r)
	}
}

// Ensure all jobs pushed are immediately handled and executed concurrently.
func TestAsyncQueue_Basic(t *testing.T) {
	queue := New(AsyncMode)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	done := make(chan struct{}, 1)
	go func() {
		err := queue.Start(ctx)
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}
		done <- struct{}{}
	}()

	results := make([]string, 0, 3)
	makeJob := func(id string) Runner {
		return basicTestJob(func() error {
			time.Sleep(time.Second)
			results = append(results, id)
			return nil
		})
	}

	waitUntilStarted(t, queue)
	id, err := queue.Push(ctx, makeJob("job1"))
	check(t, 1, id, err)
	id, err = queue.Push(ctx, makeJob("job2"))
	check(t, 2, id, err)
	id, err = queue.Push(ctx, makeJob("job3"))
	check(t, 3, id, err)

	time.Sleep(1500 * time.Millisecond)

	if err := queue.Stop(ctx); err != nil {
		t.Errorf("expected no error but got %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("running queue did not exit.")
	}

	if lg := len(results); lg != 3 {
		t.Fatalf("invalid results length. expected 3 but got %d", lg)
		return
	}
	expect := []string{"job1", "job2", "job3"}
	resultsStr := strings.Join(results, " ")
	for _, e := range expect {
		if !strings.Contains(resultsStr, e) {
			t.Errorf("expected %v but got %v", expect, results)
			return
		}
	}
}

// Ensure pending job into the internal queue will not be executed once the Queue is stopped.
func TestSyncQueue_StopOngoing(t *testing.T) {
	queue := New(SyncMode)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	done := make(chan struct{}, 1)
	go func() {
		err := queue.Start(ctx)
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}
		done <- struct{}{}
	}()

	results := make([]string, 0, 2)
	makeJob := func(id string) Runner {
		return basicTestJob(func() error {
			time.Sleep(time.Second)
			results = append(results, id)
			return nil
		})
	}

	waitUntilStarted(t, queue)
	id, err := queue.Push(ctx, makeJob("job1"))
	check(t, 1, id, err)
	id, err = queue.Push(ctx, makeJob("job2"))
	check(t, 2, id, err)
	id, err = queue.Push(ctx, makeJob("job3"))
	check(t, 3, id, err)

	time.Sleep(2500 * time.Millisecond)

	if err := queue.Stop(ctx); err != nil {
		t.Errorf("expected no error but got %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("running queue did not exit.")
	}

	if lg := len(results); lg != 2 {
		t.Fatalf("invalid results length. expected 2 but got %d", lg)
		return
	}

	if results[0] != "job1" {
		t.Errorf("expected %q but got %s", "job1", results[0])
	}
	if results[1] != "job2" {
		t.Errorf("expected %q but got %s", "job2", results[1])
	}
}

// Ensure pending job into the internal queue will not be executed once the Queue max uptime reached.
func TestSyncQueue_TimeoutOngoing(t *testing.T) {
	queue := New(SyncMode)
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	done := make(chan struct{}, 1)
	go func() {
		err := queue.Start(ctx)
		if err != context.DeadlineExceeded {
			t.Errorf("expected error %v but got %v", context.DeadlineExceeded, err)
		}
		done <- struct{}{}
	}()

	results := make([]string, 0, 2)
	makeJob := func(id string) Runner {
		return basicTestJob(func() error {
			time.Sleep(time.Second)
			results = append(results, id)
			return nil
		})
	}
	waitUntilStarted(t, queue)
	id, err := queue.Push(ctx, makeJob("job1"))
	check(t, 1, id, err)
	id, err = queue.Push(ctx, makeJob("job2"))
	check(t, 2, id, err)
	id, err = queue.Push(ctx, makeJob("job3"))
	check(t, 3, id, err)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Error("running queue did not exit.")
	}

	if lg := len(results); lg != 2 {
		t.Fatalf("invalid results. expected 2 items but got %d", lg)
		return
	}
	if results[0] != "job1" {
		t.Errorf("expected %q but got %s", "job1", results[0])
	}
	if results[1] != "job2" {
		t.Errorf("expected %q but got %s", "job2", results[1])
	}
}

type basicTestJob func() error

func (b basicTestJob) Run() error {
	return b()
}

// check verifies if err is nil and if id equals expect.
func check(t *testing.T, expect int64, id int64, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("expected no error but got %v", err)
	}

	if id != expect {
		t.Errorf("expected %d but got %d", expect, id)
	}
}

func waitUntilStarted(t *testing.T, queue Queuer) {
	t.Helper()
	for !queue.IsRunning() {
		time.Sleep(1 * time.Microsecond)
	}
}
