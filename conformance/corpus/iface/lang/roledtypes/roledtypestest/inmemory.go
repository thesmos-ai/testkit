// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package roledtypestest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/lang/roledtypes], and the
// in-memory subject they are run against.
package roledtypestest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/roledtypes"
)

// ErrNotFound is what Get reports for a key nothing wrote.
var ErrNotFound = errors.New("roledtypestest: not found")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu       sync.Mutex
	payloads map[roledtypes.Key]roledtypes.Payload
}

var _ roledtypes.Store = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{payloads: map[roledtypes.Key]roledtypes.Payload{}}
}

// Put writes a payload under a key.
func (s *InMemory) Put(ctx context.Context, key roledtypes.Key, payload roledtypes.Payload) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payloads[key] = payload
	return nil
}

// Get reads one back, and reports a miss with the zero value.
func (s *InMemory) Get(ctx context.Context, key roledtypes.Key) (roledtypes.Payload, error) {
	if err := contextErr(ctx); err != nil {
		return roledtypes.Payload{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.payloads[key]
	if !ok {
		return roledtypes.Payload{}, ErrNotFound
	}
	return p, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("roledtypestest: nil context")
	}
	return ctx.Err()
}
