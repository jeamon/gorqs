package goq

import (
	"errors"
)

type Flag uint8

const SYNC_QUEUE_SIZE = 32

const (
	MODE_SYNC Flag = 1 << iota
	MODE_ASYNC
	TRACK_JOBS
)

var (
	ErrPending        = errors.New("pending into queue")
	ErrRunning        = errors.New("still running")
	ErrNotFound       = errors.New("not found")
	ErrInvalid        = errors.New("found non error result")
	ErrNotImplemented = errors.New("feature not enabled. add TRACK_JOBS flag when creating the queue")
)
