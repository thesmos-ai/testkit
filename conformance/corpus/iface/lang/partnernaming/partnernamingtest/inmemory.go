// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package partnernamingtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/lang/partnernaming], and
// the in-memory subject they are run against.
package partnernamingtest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/partnernaming"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// Three effects and three observers, which is the fixture's point: the axis
// is about which of them a generator can pair from the source alone, and a
// subject is what makes the pairing it did write checkable.
type InMemory struct {
	mu      sync.Mutex
	weights map[string]int
	at      map[string]int
	buckets map[int]int
}

var _ partnernaming.Store = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{
		weights: map[string]int{},
		at:      map[string]int{},
		buckets: map[int]int{},
	}
}

// Touch adds the weight to what the key carries.
func (s *InMemory) Touch(ctx context.Context, key string, weight int) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.weights[key] += weight
	return nil
}

// Seen reports what Touch left behind.
func (s *InMemory) Seen(ctx context.Context, k string) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.weights[k], nil
}

// Move takes one from the source and gives it to the destination.
func (s *InMemory) Move(ctx context.Context, from, to string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.at[from]--
	s.at[to]++
	return nil
}

// At reports what lives at one end of a move.
func (s *InMemory) At(ctx context.Context, where string) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.at[where], nil
}

// Emit files the identifier under the bucket its length names, which is the
// projection no signature reveals — the absence the fixture is about.
func (s *InMemory) Emit(ctx context.Context, id string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buckets[len(id)]++
	return nil
}

// Count reports how many records a bucket holds.
func (s *InMemory) Count(ctx context.Context, bucket int) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buckets[bucket], nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("partnernamingtest: nil context")
	}
	return ctx.Err()
}
