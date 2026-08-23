// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package ttltest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/ttl], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package ttltest

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/ttl"
)

// StaleKey is the lever a check pulls to reach the expired state.
//
// The subject owns its clock so that nothing reads the wall clock, which
// leaves a check with no way to advance it — and the expiry arm is the whole
// claim, so it cannot go unexercised. A value stored under this key is stamped
// a full lifetime in the past, which is the state an elapsed entry is in.
const StaleKey = "ttltest-stale"

// lifetime is the duration the directive declares. Spelled here as well
// because the subject has to honour the number the declaration states, and a
// subject that expired on a different one would pass its own tests and fail
// the claim.
const lifetime = time.Minute

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	clk    clock.Clock
	values map[string]entry
}

type entry struct {
	value ttl.Value
	at    time.Time
}

var _ ttl.Mixed = (*InMemory)(nil)

// NewInMemoryOn returns a store reading the supplied clock — the door the
// generated ModelClocked option opens, so the aging laws advance exactly the
// time this subject reads.
func NewInMemoryOn(clk clock.Clock) *InMemory {
	s := NewInMemory()
	s.clk = clk
	return s
}

// NewInMemory returns a store on a clock that only moves when told to.
func NewInMemory() *InMemory {
	return &InMemory{
		clk:    clock.NewTestClock(time.Unix(0, 0).UTC()),
		values: map[string]entry{},
	}
}

// Put stores the value and stamps when its lifetime began.
func (s *InMemory) Put(ctx context.Context, v ttl.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	at := s.clk.Now()
	if v.Key == StaleKey {
		at = at.Add(-lifetime)
	}
	s.values[v.Key] = entry{value: v, at: at}
	return nil
}

// Read returns the value while its lifetime holds, and the declared sentinel
// once it has elapsed.
func (s *InMemory) Read(ctx context.Context, key string) (ttl.Value, error) {
	if err := contextErr(ctx); err != nil {
		return ttl.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.values[key]
	if !ok {
		return ttl.Value{}, ttl.ErrExpired
	}
	if s.clk.Now().Sub(e.at) >= lifetime {
		return ttl.Value{}, ttl.ErrExpired
	}
	return e.value, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("ttltest: nil context")
	}
	return ctx.Err()
}

// Reopen is the crash seam: a store over the medium prior left behind.
//
// No Close first, and that is the whole point — a crash is the goodbye
// that never happens, so nothing gets flushed on the way out. What
// survives is what was already in the map, which is this fake's medium
// in the same sense a file is a real store's.
//
// A fresh mutex over a shared map, because the lock belongs to the
// process and the map does not. Only one incarnation is live at a time,
// so the two never contend.
func Reopen(prior *InMemory) *InMemory {
	return &InMemory{clk: prior.clk, values: prior.values}
}
