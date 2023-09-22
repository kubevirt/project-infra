package api

import (
	"context"
	"time"
)

type StartedStatus struct {
	Timestamp int
	Repos     map[string]string
}

type Change interface {
	Matches(*StartedStatus) bool
	ID() int
}

type Query interface {
	Query(ctx context.Context, startOfReport time.Time, endOfReport time.Time) ([]Change, error)
}
