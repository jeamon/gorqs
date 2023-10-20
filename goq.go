package goq

import (
	"context"
	"fmt"
	"sync/atomic"
)

type Mode uint

const (
	SYNC_MODE Mode = iota
	ASYNC_MODE
)

type Jobber interface {
	Run() error
}

type Queuer interface {
	Start(ctx context.Context) error
	Push(ctx context.Context, j Jobber) error
	Stop(ctx context.Context) error
}

type Queue struct {
	jobsChan chan Jobber
	stopped  atomic.Bool
	mode     Mode
}

func New(m Mode) *Queue {
	return &Queue{
		jobsChan: make(chan Jobber),
		mode:     m,
	}
}

// Start pulls job from the queue and runs them.
// It returns once the context is cancelled.
func (q *Queue) Start(ctx context.Context) error {
	switch q.mode {
	case SYNC_MODE:
		return q.syncStart(ctx)
	case ASYNC_MODE:
		return q.asyncStart(ctx)
	default:
		return fmt.Errorf("invalid mode")
	}
}

// syncStart uses a single worker with an internal buffered channel
// to queue and process in order received jobs. The internal queue
// size is 512. One job is processed at a given time.
func (q *Queue) syncStart(ctx context.Context) error {
	iq := make(chan Jobber, 512)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case j := <-iq:
				_ = j.Run()
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
// It returns once the context is cancelled.
func (q *Queue) asyncStart(ctx context.Context) error {
	for {
		select {
		case j := <-q.jobsChan:
			go func() { _ = j.Run() }()
		case <-ctx.Done():
			return ctx.Err()
		default:
			if q.stopped.Load() {
				return nil
			}
		}
	}
}

// Push adds a job to the queue for processing into a non-blocking fashion.
// If the queue is closed, it should retry until the context is cancelled.
func (q *Queue) Push(ctx context.Context, j Jobber) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if !q.stopped.Load() {
				q.jobsChan <- j
				return nil
			}
		}
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
