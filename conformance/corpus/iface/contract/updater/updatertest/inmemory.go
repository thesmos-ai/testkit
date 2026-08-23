// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package updatertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/updater], and the
// in-memory subject they are run against.
package updatertest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/updater"
)

// ErrNotFound reports a key the store does not hold.
var ErrNotFound = errors.New("updatertest: no value under that key")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// The write count is kept per key because `AUTO-UPDATER-REPLACES` is a claim
// about what a second write does to the first: a store appending revisions
// returns the newest on a read and satisfies every comparison of one value, and
// grows without bound.
type InMemory struct {
	mu     sync.Mutex
	values map[string]updater.Value
	writes map[string]int
}

var _ updater.Contract = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{
		values: map[string]updater.Value{},
		writes: map[string]int{},
	}
}

// Put replaces whatever the key held, and establishes it when it held nothing.
//
// Establishing rather than refusing. A store that demanded a prior value would
// have no first write, and the update contract is about what happens to the old
// value rather than about who may write.
func (s *InMemory) Put(ctx context.Context, v updater.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
	s.writes[v.Key]++
	return nil
}

// Get returns the zero value alongside every error it reports, so a caller who
// checks the error and one who checks the value do not disagree about whether
// the call succeeded.
func (s *InMemory) Get(ctx context.Context, key string) (updater.Value, error) {
	if err := contextErr(ctx); err != nil {
		return updater.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, present := s.values[key]
	if !present {
		return updater.Value{}, ErrNotFound
	}
	return v, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("updatertest: nil context")
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
	return &InMemory{values: prior.values, writes: prior.writes}
}
