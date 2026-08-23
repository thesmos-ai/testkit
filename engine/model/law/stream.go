// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law

import (
	"fmt"
	"sort"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/engine/model/history"
)

// StreamReentrancy checks that iterating a StreamReader-shaped method
// twice produces the same items. Catches one-shot iterators or
// iterators that mutate state during iteration.
//
// Iteration is observational — the stream should not modify state.
type StreamReentrancy[T any, V any] struct {
	// Collect drains the stream iterator into a slice.
	Collect func(*rapid.T, T) ([]V, error)
}

// ID returns the stable identifier for this law.
func (StreamReentrancy[T, V]) ID() string { return lawid.StreamReentrant }

// REQID returns an empty string (auto-derived law).
func (StreamReentrancy[T, V]) REQID() string { return "" }

// Check verifies reentrancy by collecting the stream twice and
// comparing the results (order-insensitive).
func (l StreamReentrancy[T, V]) Check(rt *rapid.T, sut, _ T) error {
	first, err1 := l.Collect(rt, sut)
	if err1 != nil {
		return fmt.Errorf("StreamReentrancy: first iteration error: %w", err1)
	}
	second, err2 := l.Collect(rt, sut)
	if err2 != nil {
		return fmt.Errorf("StreamReentrancy: second iteration error: %w", err2)
	}
	sortByString(first)
	sortByString(second)
	if diff := cmp.Diff(first, second); diff != "" {
		return fmt.Errorf("StreamReentrancy: iterations differ (-first +second):\n%s", diff)
	}
	return nil
}

// sortByString sorts a slice by the Sprint representation of each element.
func sortByString[V any](s []V) {
	sort.Slice(s, func(i, j int) bool {
		return fmt.Sprint(s[i]) < fmt.Sprint(s[j])
	})
}

// StreamCompletion verifies a Stream-class method terminates within
// a finite consumer-supplied limit. Auto-emitted for every
// StreamReader (limit defaults to 10000 in the runner).
type StreamCompletion[T any, V any] struct {
	Drain func(*rapid.T, T) ([]V, error)
	// Limit bounds the drain so a non-terminating stream fails rather than
	// hangs. Zero defaults to 10000, which is the claim: a drain that needs
	// more than that has not terminated in any sense a test can wait for.
	Limit int
}

// ID returns the stable identifier for this law.
func (StreamCompletion[T, V]) ID() string { return lawid.StreamCompletion }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (StreamCompletion[T, V]) REQID() string { return "" }

// Check verifies the drain produced fewer than Limit items.
func (l StreamCompletion[T, V]) Check(rt *rapid.T, sut, _ T) error {
	items, err := l.Drain(rt, sut)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	limit := l.Limit
	if limit <= 0 {
		limit = 10000
	}
	if len(items) >= limit {
		return fmt.Errorf("StreamCompletion: drain reached limit %d without terminating", limit)
	}
	return nil
}

// StreamNoDuplicates verifies a Stream-class method emits each
// element at most once per drain (under the consumer-supplied
// hash function). Auto-emitted for Streams whose contract excludes
// duplicates (Paginator, set-typed Stream, etc.).
type StreamNoDuplicates[T any, V any, H comparable] struct {
	Drain func(*rapid.T, T) ([]V, error)
	Hash  func(V) H
}

// ID returns the stable identifier for this law.
func (StreamNoDuplicates[T, V, H]) ID() string { return lawid.StreamNoDuplicates }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (StreamNoDuplicates[T, V, H]) REQID() string { return "" }

// Check verifies no element hash appears twice.
func (l StreamNoDuplicates[T, V, H]) Check(rt *rapid.T, sut, _ T) error {
	items, err := l.Drain(rt, sut)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	seen := make(map[H]struct{}, len(items))
	for i, v := range items {
		h := l.Hash(v)
		if _, dup := seen[h]; dup {
			return fmt.Errorf("StreamNoDuplicates: element %d duplicates an earlier value (hash=%v)", i, h)
		}
		seen[h] = struct{}{}
	}
	return nil
}

// StreamStableOrder verifies a Stream-class method emits elements
// in an order consistent with the consumer-supplied total order
// (e.g., insertion order, key-ascending). Auto-emitted for Streams
// carrying //testkit:mixin stableorder.
type StreamStableOrder[T any, V any] struct {
	Drain func(*rapid.T, T) ([]V, error)
	Less  func(a, b V) bool
}

// ID returns the stable identifier for this law.
func (StreamStableOrder[T, V]) ID() string { return lawid.StreamStableOrder }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (StreamStableOrder[T, V]) REQID() string { return "" }

// Check verifies the drained order is sorted by Less.
func (l StreamStableOrder[T, V]) Check(rt *rapid.T, sut, _ T) error {
	items, err := l.Drain(rt, sut)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	for i := 1; i < len(items); i++ {
		if l.Less(items[i], items[i-1]) {
			return fmt.Errorf("StreamStableOrder: element %d precedes %d in order", i, i-1)
		}
	}
	return nil
}

// StreamPermutation verifies a Stream's drain is a permutation of what
// the run wrote. Auto-emitted for Streams carrying //testkit:mixin
// permutation.
//
// The expectation is the [history.History] the recording write fills,
// not a closure over the subject. It was one, and a closure over the
// subject cannot answer this: what a collection SHOULD hold is what went
// into it, and the action stream is the only party that knows. Asked for
// it, a consumer had a choice between reading the drain again — which
// makes the law a tautology — and keeping a second copy of the state by
// hand, for a question the run already has the answer to.
type StreamPermutation[T any, V any, H comparable] struct {
	Drain   func(*rapid.T, T) ([]V, error)
	History *history.History[string, V]
	Hash    func(V) H
}

// ID returns the stable identifier for this law.
func (StreamPermutation[T, V, H]) ID() string { return lawid.StreamPermutation }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (StreamPermutation[T, V, H]) REQID() string { return "" }

// Check verifies got is a permutation of what was written (by hash).
func (l StreamPermutation[T, V, H]) Check(rt *rapid.T, sut, _ T) error {
	got, err := l.Drain(rt, sut)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	expected := l.History.Snapshot("")
	if len(got) != len(expected) {
		return fmt.Errorf("StreamPermutation: drained %d items, expected %d", len(got), len(expected))
	}
	gotH := make([]H, len(got))
	for i, v := range got {
		gotH[i] = l.Hash(v)
	}
	expH := make([]H, len(expected))
	for i, v := range expected {
		expH[i] = l.Hash(v)
	}
	sort.Slice(gotH, func(i, j int) bool { return fmt.Sprint(gotH[i]) < fmt.Sprint(gotH[j]) })
	sort.Slice(expH, func(i, j int) bool { return fmt.Sprint(expH[i]) < fmt.Sprint(expH[j]) })
	for i := range gotH {
		if gotH[i] != expH[i] {
			return fmt.Errorf("StreamPermutation: sorted-hash mismatch at %d: got=%v expected=%v", i, gotH[i], expH[i])
		}
	}
	return nil
}

// StreamOverMatch verifies the SUT's stream drain holds everything the
// run wrote, and permits it to hold more. Auto-emitted for Streams
// carrying //testkit:mixin overmatch.
//
// The required set is the [history.History] the recording write fills,
// for the reason [StreamPermutation] gives: what a drain OWES is what
// went into it, and the action stream is the only party that knows.
type StreamOverMatch[T any, V any, H comparable] struct {
	Drain   func(*rapid.T, T) ([]V, error)
	History *history.History[string, V]
	Hash    func(V) H
}

// ID returns the stable identifier for this law.
func (StreamOverMatch[T, V, H]) ID() string { return lawid.StreamOverMatch }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (StreamOverMatch[T, V, H]) REQID() string { return "" }

// Check verifies the drain contains every required element.
func (l StreamOverMatch[T, V, H]) Check(rt *rapid.T, sut, _ T) error {
	got, err := l.Drain(rt, sut)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	seen := make(map[H]struct{}, len(got))
	for _, v := range got {
		seen[l.Hash(v)] = struct{}{}
	}
	for _, want := range l.History.Snapshot("") {
		if _, ok := seen[l.Hash(want)]; !ok {
			return fmt.Errorf("StreamOverMatch: written element %v missing from drain", want)
		}
	}
	return nil
}

// StreamReflectsMutations verifies a Stream-class method reflects
// prior mutations: an item written via Put appears in the next
// drain, and (when Delete is supplied) a deleted item disappears
// from it. Auto-emitted for the
// //testkit:mixin streamreflectsmutations mutate=<M> delete=<M> directive.
//
// Delete may be nil for interfaces without a deleter; the law then
// checks only the put direction. A stream serving a stale snapshot
// fails on the first put.
type StreamReflectsMutations[T any, V any, H comparable] struct {
	Put    func(*rapid.T, T, V) error
	Delete func(*rapid.T, T, V) error
	Drain  func(*rapid.T, T) ([]V, error)
	Values *rapid.Generator[V]
	Hash   func(V) H
}

// ID returns the stable identifier for this law.
func (StreamReflectsMutations[T, V, H]) ID() string { return lawid.StreamReflectsMutations }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (StreamReflectsMutations[T, V, H]) REQID() string { return "" }

// count reports how many drained elements hash equal to v.
//
// A count rather than a membership test, because the stream may lawfully
// hold copies the run put there earlier: presence after a delete proves
// nothing when a stranger's copy remains, and absence would be a claim about
// their copy rather than this Check's own.
func (l StreamReflectsMutations[T, V, H]) count(rt *rapid.T, sut T, v V) (int, error) {
	items, err := l.Drain(rt, sut)
	if err != nil {
		return 0, err
	}
	h := l.Hash(v)
	n := 0
	for _, it := range items {
		if l.Hash(it) == h {
			n++
		}
	}
	return n, nil
}

// Check puts a value and verifies the stream gained a copy; when Delete is
// supplied, deletes it and verifies the stream returned to the count it held
// before the put.
func (l StreamReflectsMutations[T, V, H]) Check(rt *rapid.T, sut, _ T) error {
	v := l.Values.Draw(rt, "StreamReflectsMutations_value")
	before, err := l.count(rt, sut, v)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	if putErr := l.Put(rt, sut, v); putErr != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	after, err := l.count(rt, sut, v)
	if err != nil {
		return fmt.Errorf("StreamReflectsMutations: drain after put errored: %w", err)
	}
	if after <= before {
		return fmt.Errorf("StreamReflectsMutations: value %v not in stream after put (%d → %d copies)",
			v, before, after)
	}
	if l.Delete == nil {
		return nil
	}
	if delErr := l.Delete(rt, sut, v); delErr != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	restored, err := l.count(rt, sut, v)
	if err != nil {
		return fmt.Errorf("StreamReflectsMutations: drain after delete errored: %w", err)
	}
	if restored != before {
		return fmt.Errorf("StreamReflectsMutations: value %v: delete did not restore the count (%d → %d)",
			v, before, restored)
	}
	return nil
}
