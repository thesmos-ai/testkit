// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package boundedcachetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/boundedcache], and
// the in-memory subject they are run against.
package boundedcachetest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/boundedcache"
)

// InMemory is the implementation the generated conformance harness is run
// against: a keyed store that evicts the oldest entry once it is full.
//
// Oldest-first is one legal policy of several, and nothing in the
// declaration picks it. The negative control beside this package is a
// cache that evicts the newest instead, which the bound permits equally —
// a check that came to require oldest-first would redden it.
type InMemory struct {
	mu       sync.Mutex
	capacity int
	order    []boundedcache.Key
	entries  map[boundedcache.Key]boundedcache.Value
}

var _ boundedcache.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty cache holding at most the given capacity.
//
// The capacity is a parameter rather than a constant because the harness
// hands it over: `//testkit:mixin bounded limit=2` is read by the
// generator and passed to every constructor, so a subject restating the
// number could disagree with the law measuring it.
func NewInMemory(capacity int) *InMemory {
	return &InMemory{
		capacity: capacity,
		entries:  map[boundedcache.Key]boundedcache.Value{},
	}
}

// Put stores the value, evicting the oldest key when a NEW key would take
// the cache past its capacity. Overwriting a held key evicts nothing —
// the count does not move, so there is nothing to make room for.
func (s *InMemory) Put(ctx context.Context, key boundedcache.Key, value boundedcache.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, held := s.entries[key]; !held {
		if len(s.order) >= s.capacity {
			oldest := s.order[0]
			s.order = s.order[1:]
			delete(s.entries, oldest)
		}
		s.order = append(s.order, key)
	}
	s.entries[key] = value
	return nil
}

// Get answers what is held, and false for a key that was never written or
// has since been evicted. The two are indistinguishable on purpose:
// telling them apart would be a claim about the victim policy.
func (s *InMemory) Get(ctx context.Context, key boundedcache.Key) (boundedcache.Value, bool) {
	if err := contextErr(ctx); err != nil {
		return boundedcache.Value{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, held := s.entries[key]
	return v, held
}

// Len counts what is held, which the eviction in Put keeps at or under
// the capacity.
func (s *InMemory) Len(ctx context.Context) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries), nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("boundedcachetest: nil context")
	}
	return ctx.Err()
}
