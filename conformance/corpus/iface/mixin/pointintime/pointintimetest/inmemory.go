// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pointintimetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/pointintime], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package pointintimetest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/pointintime"
)

// ErrNotFound is what Get reports for a key nothing wrote.
var ErrNotFound = errors.New("pointintimetest: not found")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]pointintime.Value
}

var _ pointintime.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]pointintime.Value{}}
}

// Store records the value under its own key.
func (s *InMemory) Store(ctx context.Context, v pointintime.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
	return nil
}

// Get returns what Store recorded, and reports a miss with the zero value.
func (s *InMemory) Get(ctx context.Context, key string) (pointintime.Value, error) {
	if err := contextErr(ctx); err != nil {
		return pointintime.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	if !ok {
		return pointintime.Value{}, ErrNotFound
	}
	return v, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("pointintimetest: nil context")
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
	return &InMemory{values: prior.values}
}
