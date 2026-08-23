// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package ttlperwritetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/ttlperwrite], and the
// in-memory subject they are run against.
package ttlperwritetest

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/ttlperwrite"
)

// InMemory expires each entry at the lifetime that entry carried.
//
// The lifetime is per write, which is the whole of what separates this
// fixture from its sibling: two entries written in the same instant can
// expire at different times, and a store that took the lifetime from
// anywhere but the entry would still pass a suite where every write
// declared the same one.
type InMemory struct {
	mu      sync.Mutex
	clk     clock.Clock
	entries map[string]held
}

// held is an entry beside the instant it stops being readable.
type held struct {
	entry ttlperwrite.Entry
	until time.Time
}

var _ ttlperwrite.Mixed = (*InMemory)(nil)

// NewInMemory returns a store on a clock that only moves when told to.
func NewInMemory() *InMemory { return NewInMemoryOn(clock.NewTestClock(time.Unix(0, 0).UTC())) }

// NewInMemoryOn returns a store reading the clock the run controls.
func NewInMemoryOn(clk clock.Clock) *InMemory {
	return &InMemory{clk: clk, entries: map[string]held{}}
}

// Put stores an entry, starting the lifetime it carries.
//
// A zero lifetime means forever, which is the reading that makes
// stripping the field a defect rather than a refusal: the write lands,
// the read answers, and the only thing that changed is that nothing
// expires.
func (s *InMemory) Put(ctx context.Context, e ttlperwrite.Entry) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h := held{entry: e}
	if e.Lifetime > 0 {
		h.until = s.clk.Now().Add(e.Lifetime)
	}
	s.entries[e.Key] = h
	return nil
}

// Read answers while the entry's own lifetime holds.
func (s *InMemory) Read(ctx context.Context, key string) (ttlperwrite.Entry, error) {
	if err := contextErr(ctx); err != nil {
		return ttlperwrite.Entry{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h, present := s.entries[key]
	switch {
	case !present:
		return ttlperwrite.Entry{}, ttlperwrite.ErrExpired
	case !h.until.IsZero() && !s.clk.Now().Before(h.until):
		delete(s.entries, key)
		return ttlperwrite.Entry{}, ttlperwrite.ErrExpired
	}
	return h.entry, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("ttlperwritetest: nil context")
	}
	return ctx.Err()
}

// Reopen is the crash seam: a store over the medium prior left behind.
//
// The clock carries across with the map. A rebuild that started time
// again from zero would make every entry look freshly written, and the
// expiry claim would hold for a store that had lost the deadline.
func Reopen(prior *InMemory) *InMemory {
	return &InMemory{clk: prior.clk, entries: prior.entries}
}
