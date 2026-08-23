// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package validatestest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/validates], and the in-memory
// subject they are run against.
package validatestest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/validates"
)

// ErrNotFound is what Read reports for a key nothing stored.
var ErrNotFound = errors.New("validatestest: not found")

// ErrInvalid is what Validate refuses a payload with.
var ErrInvalid = errors.New("validatestest: invalid payload")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// It lives beside the harness rather than in the package declaring the
// interface, because that is what it is for: a fixture package states a shape,
// and the subject a conformance run holds to it is scaffolding for the run.
//
// A fixture package declaring only an interface can be generated for and
// compiled, but nothing can be *run* against it — and a harness nobody runs is
// indistinguishable from one whose checks cannot fail. This is the subject that
// makes the corpus prove the difference.
//
// It is written to satisfy every check the harness derives, which is the point:
// the list below is the contract a conformance suite states, and an
// implementation that skips one of them is one the suite is supposed to reject.
type InMemory struct {
	mu    sync.Mutex
	items map[string]validates.Payload
}

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{items: map[string]validates.Payload{}} }

// Store refuses what Validate refuses, and reports a context that is done.
func (s *InMemory) Store(ctx context.Context, v validates.Payload) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := s.Validate(v); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[v.Key] = v
	return nil
}

// Validate refuses a payload with no key.
func (*InMemory) Validate(v validates.Payload) error {
	if v.Key == "" {
		return ErrInvalid
	}
	return nil
}

// Read returns the zero value alongside every error it reports, which is the
// property the reader's own check is about: a caller who checks the error and
// one who checks the value must not disagree about whether the call succeeded.
func (s *InMemory) Read(ctx context.Context, key string) (validates.Payload, error) {
	if err := contextErr(ctx); err != nil {
		return validates.Payload{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.items[key]
	if !ok {
		return validates.Payload{}, ErrNotFound
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
		return errors.New("validatestest: nil context")
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
	return &InMemory{items: prior.items}
}
