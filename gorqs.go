// Package gorqs (stands for `Go Runnable Queue Service`) provides routines to queue and execute runnable jobs and caches
// their execution result (error) for later consultation. The Queue processor can run into synchronous or asynchronous mode.
// Adding a job to the Queue service is always a non-blocking operation and returns a unique job id on success.
package gorqs

import (
	"context"
	"sync"
	"time"
)

// New creates an instance of a workable Queue.
func New(flags Flag) *Queue {
	q := &Queue{
		jobsChan: make(chan jobber, 1),
		records:  sync.Map{},
	}

	if flags&AsyncMode != 0 {
		q.mode = AsyncMode
	} else {
		q.mode = SyncMode
	}

	if flags&TrackJobs != 0 {
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
	case SyncMode:
		return q.sstart(ctx)
	case AsyncMode:
		return q.astart(ctx)
	case TrackJobs:
		return ErrInvalidMode
	default:
		return ErrUnknownMode
	}
}

// sconsumer is a synchronous worker which process one job
// at a time. Then records the result of job processing.
func (q *Queue) sconsumer(ctx context.Context, iq *slist) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !q.running.Load() {
				return
			}
			if iq.isEmpty() {
				continue
			}
			if job := iq.pop(); job != nil {
				q.recordFn(job.getID(), ErrRunning)
				q.recordFn(job.getID(), job.Run())
			}
		}
	}
}

// sstart starts a single worker named sconsumer and pushes each received job
// onto an internal synchronized singly linked list named iq. This ensure that
// one job is processed at a time and jobs are processed in order of reception.
func (q *Queue) sstart(ctx context.Context) error {
	q.running.Store(true)
	iq := list()
	go q.sconsumer(ctx, iq)

	for {
		select {
		case <-ctx.Done():
			q.running.Store(false)
			return ctx.Err()
		case job := <-q.jobsChan:
			iq.push(job)
		default:
			if !q.running.Load() {
				return nil
			}
		}
	}
}

// astart pulls jobs from the queue and runs them asynchronously.
// It returns once the Queue is stopped or the context is done.
// Each job returned error result is stored into the records map.
func (q *Queue) astart(ctx context.Context) error {
	q.running.Store(true)
	for {
		select {
		case j := <-q.jobsChan:
			go func() {
				q.recordFn(j.getID(), ErrRunning)
				q.recordFn(j.getID(), j.Run())
			}()
		case <-ctx.Done():
			q.running.Store(false)
			return ctx.Err()
		default:
			if !q.running.Load() {
				return nil
			}
		}
	}
}

// Push is a non-blocking method that adds job to the queue for processing.
// It allows a maximum of 10ms to enqueue a job. On success, it returns the
// unique job id (int64) and nil as error. Then records into the cache an
// initial state of the job result as ErrPending.
// If the queue service is stopped, it returns ErrQueueClosed. If after 10ms
// the the job is not enqueued it returns ErrQueueBusy. In case the context
// is done or fail to enqueue the job, it ensures the job id is not cached.
func (q *Queue) Push(ctx context.Context, r Runner) (int64, error) {
	if !q.running.Load() {
		return -1, ErrQueueClosed
	}

	id := q.counter.Add(1)
	if q.mode == SyncMode {
		q.recordFn(id, ErrPending)
	}

	select {
	case <-ctx.Done():
		q.records.Delete(id)
		return -1, ctx.Err()
	case q.jobsChan <- &job{id, r}:
		return id, nil
	case <-time.After(10 * time.Millisecond):
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
		q.running.Store(false)
		return nil
	}
}

// Clear removes all recorded job results from the cache.
func (q *Queue) Clear() {
	q.records.Range(func(key interface{}, value interface{}) bool {
		q.records.Delete(key)
		return true
	})
}

// IsRunning returns the current status of the queue service.
func (q *Queue) IsRunning() bool {
	return q.running.Load()
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
func (q *Queue) result(_ context.Context, id int64) error {
	v, found := q.records.LoadAndDelete(id)
	if !found {
		return ErrNotFound
	}
	if v == nil {
		return nil
	}
	err, found := v.(error)
	if !found {
		return ErrInvalid
	}

	return err
}
