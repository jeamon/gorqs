package gorqs

import "errors"

var (
	ErrPending        = errors.New("pending into queue")
	ErrRunning        = errors.New("still running")
	ErrNotFound       = errors.New("not found")
	ErrInvalid        = errors.New("found non error result")
	ErrQueueClosed    = errors.New("queue is closed")
	ErrQueueBusy      = errors.New("queue is busy")
	ErrNotImplemented = errors.New("feature not enabled. add TRACK_JOBS flag when creating the queue")
)

type Runner interface {
	Run() error
}

type Jobber interface {
	GetID() int64
	Runner
}

type Job struct {
	id int64
	r  Runner
}

func (j *Job) GetID() int64 {
	return j.id
}

func (j *Job) Run() error {
	return j.r.Run()
}
