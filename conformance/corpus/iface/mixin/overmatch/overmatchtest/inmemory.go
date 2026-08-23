// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package overmatchtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/overmatch], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package overmatchtest

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/overmatch"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu    sync.Mutex
	items []overmatch.Value
}

var _ overmatch.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty collection.
func NewInMemory() *InMemory { return &InMemory{} }

// Add keeps the element, and keeps a repeat beside the first.
//
// A bag rather than a keyed map, because that is what the mixin above this
// fixture declares: a collection that overwrote by key would drop an
// element it had accepted, and both `AUTO-STREAM-OVER-MATCH` and
// `AUTO-STREAM-PERMUTATION` are the claim that it does not.
func (s *InMemory) Add(ctx context.Context, v overmatch.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, v)
	return nil
}

// Items drains the collection in key order, ties broken by body.
//
// Sorted rather than in arrival order, and by both members rather than the
// key alone: two elements sharing a key must still come out the same way
// on every call, or a drain compared against its own second reading
// disagrees for a reason that has nothing to do with the subject.
func (s *InMemory) Items(ctx context.Context) ([]overmatch.Value, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := slices.Clone(s.items)
	slices.SortFunc(out, func(a, b overmatch.Value) int {
		if k := strings.Compare(a.Key, b.Key); k != 0 {
			return k
		}
		return strings.Compare(a.Body, b.Body)
	})
	return out, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("overmatchtest: nil context")
	}
	return ctx.Err()
}
