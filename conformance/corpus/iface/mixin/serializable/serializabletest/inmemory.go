// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package serializabletest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/serializable], and the
// in-memory subject they are run against — scaffolding for the run, so it
// lives beside the harness rather than in the package stating the shape.
package serializabletest

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/serializable"
)

// ErrNotFound is what Get reports for a key nothing recorded.
var ErrNotFound = errors.New("serializabletest: not found")

// ErrNotAdmissible is what Record reports for an operation the level
// forbids — the refusal that makes an isolation claim a policy rather
// than a description.
var ErrNotAdmissible = errors.New("serializabletest: the isolation level forbids it")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu      sync.Mutex
	entries []serializable.Entry
	txn     int
	version map[string]int64
	wrote   map[keyed]bool
}

var _ serializable.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty history.
func NewInMemory() *InMemory {
	return &InMemory{version: map[string]int64{}, wrote: map[keyed]bool{}}
}

// Record appends one operation, and refuses one that would put an anomaly
// in the history it reports.
//
// A store claiming an isolation level is an admission policy before it is
// anything else: the level IS the set of operations it will not accept.
// Three rules, all of them the domain's rather than the checkers':
//
//   - a transaction identifier and a version are assigned by the store, so
//     both are positive;
//   - transactions arrive in order and a closed one does not reopen, so a
//     record for a transaction below the highest seen is refused;
//   - a write installs a version above the key's current one and a
//     transaction writes a key once, so no write is left intermediate for a
//     later read to observe;
//   - a read observes the version a key currently holds, so nothing reads
//     what was never written or what has since been replaced.
//
// Together those make the recorded history serial by construction, which is
// the strictest reading of what this fixture declares. A store that skipped
// them would report a history whose anomalies it created itself.
func (s *InMemory) Record(ctx context.Context, e serializable.Entry) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.admits(e); err != nil {
		return err
	}
	if e.Txn > s.txn {
		s.txn = e.Txn
	}
	if e.Write {
		s.version[e.Key] = e.Version
		s.wrote[keyed{txn: e.Txn, key: e.Key}] = true
	}
	s.entries = append(s.entries, e)
	return nil
}

// keyed pairs a transaction with a key, so a second write to one key by one
// transaction is refusable without scanning the history.
type keyed struct {
	txn int
	key string
}

// admits is the policy, stated once so Record reads as what it does.
func (s *InMemory) admits(e serializable.Entry) error {
	switch {
	case e.Txn <= 0:
		return fmt.Errorf("%w: transaction %d", ErrNotAdmissible, e.Txn)
	case e.Version <= 0:
		return fmt.Errorf("%w: version %d", ErrNotAdmissible, e.Version)
	case e.Txn < s.txn:
		return fmt.Errorf("%w: transaction %d closed at %d", ErrNotAdmissible, e.Txn, s.txn)
	case e.Write && e.Version <= s.version[e.Key]:
		return fmt.Errorf("%w: %q holds version %d", ErrNotAdmissible, e.Key, s.version[e.Key])
	case e.Write && s.wrote[keyed{txn: e.Txn, key: e.Key}]:
		return fmt.Errorf("%w: transaction %d already wrote %q", ErrNotAdmissible, e.Txn, e.Key)
	case !e.Write && e.Version != s.version[e.Key]:
		return fmt.Errorf("%w: %q holds version %d, not %d",
			ErrNotAdmissible, e.Key, s.version[e.Key], e.Version)
	}
	return nil
}

// History reports the recorded operations.
//
// A copy rather than the slice itself: an anomaly check walks the history
// while the subject may still be recording, and handing out the backing array
// would let a later append be observed mid-walk.
func (s *InMemory) History(ctx context.Context) ([]serializable.Entry, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.entries), nil
}

// Get answers the latest recorded entry for a key, and reports a miss with
// the zero value — the read half the anomaly law instantiates its key at.
func (s *InMemory) Get(ctx context.Context, key string) (serializable.Entry, error) {
	if err := contextErr(ctx); err != nil {
		return serializable.Entry{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range slices.Backward(s.entries) {
		if e.Key == key {
			return e, nil
		}
	}
	return serializable.Entry{}, ErrNotFound
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("serializabletest: nil context")
	}
	return ctx.Err()
}
