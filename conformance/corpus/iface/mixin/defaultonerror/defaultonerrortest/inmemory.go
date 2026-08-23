// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package defaultonerrortest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/defaultonerror], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package defaultonerrortest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/defaultonerror"
)

// ErrNotFound is what Get reports for a key nothing wrote.
var ErrNotFound = errors.New("defaultonerrortest: not found")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]defaultonerror.Value
}

var _ defaultonerror.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]defaultonerror.Value{}}
}

// Store records the value under its own key.
func (s *InMemory) Store(ctx context.Context, v defaultonerror.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
	return nil
}

// Get returns what Store recorded, and reports a miss with the zero value —
// which is the default this shape answers an absent key with.
func (s *InMemory) Get(ctx context.Context, key string) (defaultonerror.Value, error) {
	if err := contextErr(ctx); err != nil {
		return defaultonerror.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	if !ok {
		return defaultonerror.Value{}, ErrNotFound
	}
	return v, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("defaultonerrortest: nil context")
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
