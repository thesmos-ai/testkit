// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package persistertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/persister], and the
// in-memory subject they are run against.
package persistertest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/persister"
)

// ErrNotFound reports a key the store does not hold.
var ErrNotFound = errors.New("persistertest: no value under that key")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// The whole value is stored rather than the body, because
// `AUTO-PERSISTER-RETRIEVABLE` compares what came back against what went in. A
// store keeping only the fields it happened to need passes a check written
// around those fields and loses the rest.
type InMemory struct {
	mu     sync.Mutex
	values map[string]persister.Value
}

var _ persister.Contract = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{values: map[string]persister.Value{}} }

// Put stores a value under its key.
func (s *InMemory) Put(ctx context.Context, v persister.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
	return nil
}

// Get returns the zero value alongside every error it reports, so a caller who
// checks the error and one who checks the value do not disagree about whether
// the call succeeded.
func (s *InMemory) Get(ctx context.Context, key string) (persister.Value, error) {
	if err := contextErr(ctx); err != nil {
		return persister.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, present := s.values[key]
	if !present {
		return persister.Value{}, ErrNotFound
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
		return errors.New("persistertest: nil context")
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
