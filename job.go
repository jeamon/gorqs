package gorqs

import "errors"

var (
	// Predefined errors
	ErrPending        = errors.New("gorqs: pending into queue")
	ErrRunning        = errors.New("gorqs: job still running")
	ErrNotFound       = errors.New("gorqs: job not found")
	ErrInvalid        = errors.New("gorqs: found non error result")
	ErrQueueClosed    = errors.New("gorqs: queue is closed")
	ErrTimeout        = errors.New("gorqs: operation took too long")
	ErrUnknownMode    = errors.New("gorqs: unknown queue mode flag")
	ErrInvalidMode    = errors.New("gorqs: invalid queue mode flag")
	ErrNotImplemented = errors.New("gorqs: feature not enabled. add TrackJobs flag when creating the queue")
)

// Runner represents a runnable job expected by the queue service.
type Runner interface {
	Run() error
}

// jobber defines expected real job behaviors.
type jobber interface {
	getID() int64
	Runner
}

// job represents the concrete item that will be pushed and processed the by the queue service.
type job struct {
	id int64
	r  Runner
}

// getID returns a given job unique id.
func (j *job) getID() int64 {
	return j.id
}

// Run implements the Run method of Runner interface.
func (j *job) Run() error {
	return j.r.Run()
}
