// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package restrictedpooltest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/restrictedpool], and
// the in-memory subject they are run against.
package restrictedpooltest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/restrictedpool"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	bodies map[restrictedpool.Key]restrictedpool.Body
}

var _ restrictedpool.Store = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{bodies: map[restrictedpool.Key]restrictedpool.Body{}}
}

// Put writes a body under a key, whatever the body is.
//
// Faithful on purpose: this is the subject the DERIVED pool runs against,
// where nobody has said what the store takes and the tier is free to probe
// with anything. A store with a domain of its own is the test file's,
// beside the run that narrows the pool to match it.
func (s *InMemory) Put(ctx context.Context, key restrictedpool.Key, body restrictedpool.Body) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bodies[key] = body
	return nil
}

// Get reads one back, and reports a miss with the declaration's sentinel.
func (s *InMemory) Get(ctx context.Context, key restrictedpool.Key) (restrictedpool.Body, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	body, ok := s.bodies[key]
	if !ok {
		return "", restrictedpool.ErrNotFound
	}
	return body, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("restrictedpooltest: nil context")
	}
	return ctx.Err()
}
