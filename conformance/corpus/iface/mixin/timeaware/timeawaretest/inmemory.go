// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package timeawaretest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/timeaware], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package timeawaretest

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/timeaware"
)

// ErrUnseen is what AgeOf reports for a key Touch never recorded.
var ErrUnseen = errors.New("timeawaretest: key never touched")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// It holds a controlled clock rather than reading the wall clock: an age
// derived from time.Now moves between the two calls of any check that reads
// it twice, and the failure would be the test's rather than the subject's.
type InMemory struct {
	mu   sync.Mutex
	clk  clock.Clock
	seen map[string]time.Time
}

var _ timeaware.Mixed = (*InMemory)(nil)

// NewInMemoryOn returns a store reading the clock the run controls.
//
// The constructor the clocked check needs. A store that read the wall
// clock would pass every check whose test never advances far enough to
// notice, and fail the one claim `timeaware` states on its own: what it
// reports has to move when the run moves time.
func NewInMemoryOn(clk clock.Clock) *InMemory {
	return &InMemory{clk: clk, seen: map[string]time.Time{}}
}

// NewInMemory returns a store on a clock that only moves when told to.
func NewInMemory() *InMemory {
	return &InMemory{
		clk:  clock.NewTestClock(time.Unix(0, 0).UTC()),
		seen: map[string]time.Time{},
	}
}

// Touch records the key at the clock's current reading.
func (s *InMemory) Touch(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[key] = s.clk.Now()
	return nil
}

// AgeOf reports the elapsed nanoseconds since the key was touched.
func (s *InMemory) AgeOf(ctx context.Context, key string) (int64, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	at, ok := s.seen[key]
	if !ok {
		return 0, ErrUnseen
	}
	return s.clk.Now().Sub(at).Nanoseconds(), nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("timeawaretest: nil context")
	}
	return ctx.Err()
}
