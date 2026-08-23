// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package answeringwritertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/answeringwriter],
// and the in-memory subject they are run against.
package answeringwritertest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/answeringwriter"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// A keyed map, and the write answers what it stored: the detector this
// fixture is named for draws on the same named type going in and coming
// back, and a subject answering something else would be a different shape.
type InMemory struct {
	mu     sync.Mutex
	values map[string]answeringwriter.Value
}

var _ answeringwriter.AnsweringWriter = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]answeringwriter.Value{}}
}

// Put stores the value and answers the stored state.
func (s *InMemory) Put(ctx context.Context, v answeringwriter.Value) (answeringwriter.Value, error) {
	if err := contextErr(ctx); err != nil {
		return answeringwriter.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
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
		return errors.New("answeringwritertest: nil context")
	}
	return ctx.Err()
}
