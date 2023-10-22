package goq

import (
	"context"
	"sync"
	"sync/atomic"
)

// Queuer defines operations on a Queue.
type Queuer interface {
	Start(ctx context.Context) error
	Push(ctx context.Context, r Runner) (int64, error)
	Stop(ctx context.Context) error
	Result(ctx context.Context, id int64) error
	Clear()
}

// Queue implements Queuer interface.
type Queue struct {
	jobsChan chan Jobber
	records  sync.Map
	stopped  atomic.Bool
	mode     Flag
	size     int
	counter  atomic.Int64
	recordFn func(id int64, err error)
}
