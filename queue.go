package gorqs

import (
	"context"
	"sync"
	"sync/atomic"
)

// Flag describes desired capabilities from the queue service at creation.
type Flag uint8

// SYNC_QUEUE_SIZE default queue buffer size when using the queue into synchronous fashion.
const SYNC_QUEUE_SIZE = 32

const (
	// Allows Queue to process incoming jobs in order of arrival and one at a time.
	MODE_SYNC Flag = 1 << iota

	// Allows Queue to process incoming jobs immediately so asynchronously.
	MODE_ASYNC

	// Allows to cache jobs execution results for further consultation.
	TRACK_JOBS
)

// Queuer defines possible operations on a Queue.
type Queuer interface {
	// Start opens the Queue to accept jobs and triggers the worker routine.
	Start(ctx context.Context) error

	// Push is a non-blocking operation to add a job wrapped around the passed Runner.
	// On success it returns a unique job id (int64 type) to fetch the job result.
	Push(ctx context.Context, r Runner) (int64, error)

	// Stop closes the Queue. On success no more job could be queued and pending jobs
	// will not be picked for execution.
	Stop(ctx context.Context) error

	// Result provides the cached result of a Runner associated to a given job id.
	// ErrNotFound is returned if the job `id` does not exist. ErrPending if the
	// job runner did not start yet. ErrRunning if picked but still being running.
	// If the job `error` result is fetched, the entry is removed from the cache.
	Result(ctx context.Context, id int64) error

	// Clear removes all executed jobs results from the records cache.
	Clear()
}

// Queue implements the Queuer interface. Use the package `New` method to get an instance.
type Queue struct {
	jobsChan chan Jobber
	records  sync.Map
	stopped  atomic.Bool
	mode     Flag
	counter  atomic.Int64
	recordFn func(id int64, err error)
	resultFn func(ctx context.Context, id int64) error
}
