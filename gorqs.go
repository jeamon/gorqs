// Package gorqs (stands for `Go Runnable Queue Service`) provides routines to queue and execute runnable jobs and caches
// their execution result for later consultation. The Queue processor can run into synchronous or asynchronous mode.
// Adding a job to the Queue service is always a non-blocking operation and returns a unique job id on success.
package gorqs

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// New creates an instance of a workable Queue.
func New(flags Flag, size, tracks int) *Queue {
	q := &Queue{
		jobsChan: make(chan Jobber, 1),
		records:  sync.Map{},
		size:     size,
	}

	if flags&MODE_ASYNC != 0 {
		q.mode = MODE_ASYNC
	} else {
		q.mode = MODE_SYNC
	}

	if q.mode == MODE_SYNC && size < SYNC_QUEUE_SIZE {
		q.size = SYNC_QUEUE_SIZE
	}

	if flags&TRACK_JOBS != 0 {
		q.recordFn = func(id int64, err error) {
			q.records.Store(id, err)
		}
		q.resultFn = func(ctx context.Context, id int64) error {
			return q.result(ctx, id)
		}
	} else {
		q.recordFn = func(id int64, err error) {}
		q.resultFn = func(ctx context.Context, id int64) error { return ErrNotImplemented }
	}
	return q
}

// Start pulls job from the queue and runs them.
// It returns once the context is cancelled.
func (q *Queue) Start(ctx context.Context) error {
	switch q.mode {
	case MODE_SYNC:
		return q.syncStart(ctx)
	case MODE_ASYNC:
		return q.asyncStart(ctx)
	default:
		return fmt.Errorf("invalid mode")
	}
}

// syncStart uses a single worker with an internal buffered channel
// to queue and process in order received jobs. The internal queue
// size is SYNC_QUEUE_SIZE. One job is processed at a time.
func (q *Queue) syncStart(ctx context.Context) error {
	iq := make(chan Jobber, q.size)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case j := <-iq:
				q.recordFn(j.GetID(), ErrRunning)
				q.recordFn(j.GetID(), j.Run())
			default:
				if q.stopped.Load() {
					return
				}
			}
		}
	}()

	for {
		select {
		case j := <-q.jobsChan:
			iq <- j
		case <-ctx.Done():
			return ctx.Err()
		default:
			if q.stopped.Load() {
				return nil
			}
		}
	}
}

// asyncStart pulls job from the queue and runs them.
// It returns once the Queue is stopped or the context
// is done. Each job returned error result is stored
// into the records map.
func (q *Queue) asyncStart(ctx context.Context) error {
	for {
		select {
		case j := <-q.jobsChan:
			go func() {
				q.recordFn(j.GetID(), ErrRunning)
				q.recordFn(j.GetID(), j.Run())
			}()
		case <-ctx.Done():
			return ctx.Err()
		default:
			if q.stopped.Load() {
				return nil
			}
		}
	}
}

// Push is a non-blocking method that adds job to the queue for processing.
// It allows a maximum of 1ms to enqueue a job. On success, it returns the
// unique job id (int64) and nil as error. Then records into the cache an
// initial state of the job result as `ErrPending`.
// If the queue service is stopped, it returns `ErrQueueClosed`. If the queue
// is full then it returns `ErrQueueBusy`. In case the context is done or fail
// to enqueue the job, the job id is would not be present into the results cache.
func (q *Queue) Push(ctx context.Context, r Runner) (int64, error) {
	if q.stopped.Load() {
		return -1, ErrQueueClosed
	}

	id := q.counter.Add(1)
	if q.mode == MODE_SYNC {
		q.recordFn(id, ErrPending)
	}

	select {
	case <-ctx.Done():
		q.records.Delete(id)
		return -1, ctx.Err()
	case q.jobsChan <- &Job{id, r}:
		return id, nil
	case <-time.After(1 * time.Millisecond):
		q.records.Delete(id)
		return -1, ErrQueueBusy
	}
}

// Stop closes the queue so no more job can be added.
func (q *Queue) Stop(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		q.stopped.Store(true)
		close(q.jobsChan)
		return nil
	}
}

// Clear removes all records from the map.
func (q *Queue) Clear() {
	q.records.Range(func(key interface{}, value interface{}) bool {
		q.records.Delete(key)
		return true
	})
}

// Result provides the result `error` of a given Job Runner based on its ID.
// If the job id was found, it delete the record from the map. It returns
// ErrNotFound if the `id` does not exist or ErrPending if the job runner
// did not start yet. ErrRunning if picked but still being processed.
// ErrNotImplemented is returned if the tracking feature was not enabled.
func (q *Queue) Result(ctx context.Context, id int64) error {
	return q.resultFn(ctx, id)
}

// result is the internal function invoked to fetch a given job execution result
// based on its id if this feature was enabled during the Queue initialization.
func (q *Queue) result(ctx context.Context, id int64) error {
	v, found := q.records.LoadAndDelete(id)
	if !found {
		return ErrNotFound
	}

	err, found := v.(error)
	if !found {
		return ErrInvalid
	}

	return err
}
