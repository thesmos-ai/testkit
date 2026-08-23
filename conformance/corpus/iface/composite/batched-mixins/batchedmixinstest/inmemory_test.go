// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// batched-mixins carries six classifications across three methods and generates
// no check for any of them: `idempotent`, `cacheable`, `pure`, `bounded` and
// `readafterwrite` are the model tier's under ADR-0028, and `concurrent` and
// `sideeffect` name no partner to observe through.
//
// What the fixture exists for is the parsing — extra positionals are further
// mixin names, and parameters are permitted only with exactly one name — so
// what the suite adds here is the signature-derived family and the rows below,
// every one of which is statable through the interface.
//
// The declared bound reaches the subject rather than being restated by it:
// `bounded limit=50` is what the harness hands every constructor, including
// each planted defect.
package batchedmixinstest_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	batchedmixins "go.thesmos.sh/testkit/conformance/corpus/iface/composite/batched-mixins"
	"go.thesmos.sh/testkit/conformance/corpus/iface/composite/batched-mixins/batchedmixinstest"
)

// TestBatchedContract runs the generated checks and this package's own.
func TestBatchedContract(t *testing.T) {
	t.Parallel()

	batchedmixinstest.RunBatched(t, inMemory("in-memory"), batchedChecks)
}

// TestBatchedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestBatchedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	batchedmixinstest.RunBatched(t,
		inMemory("in-memory"),
		batchedmixinstest.BatchedSuite.Without(batchedmixinstest.BatchedSuite.Checks.Put.Smoke()),
	)
}

// TestBatchedChecksCanFail drives every row against its planted defect.
func TestBatchedChecksCanFail(t *testing.T) {
	t.Parallel()

	batchedmixinstest.ProveBatched(t, inMemory("in-memory"), batchedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) batchedmixinstest.BatchedHarness[*batchedmixinstest.InMemory] {
	return batchedmixinstest.BatchedHarness[*batchedmixinstest.InMemory]{
		Name: name, New: batchedmixinstest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------
//
// Each row plants the near miss its own classification describes, so the
// evidence says which mixin the row is the shadow of.

var batchedChecks = batchedmixinstest.BatchedChecks{
	{
		Method: "Read", Name: "reads-back-what-put-wrote",
		Claim: "Read returns what Put wrote",
		Run:   readsBackWhatPutWrote,
		ProvenBy: batchedmixinstest.BrokenBatched(
			"a store whose reads answer the zero value", planted(answersTheZero),
		),
		ProvenReason: "carrying what was written",
	},

	{
		Method: "Put", Name: "repeat-write-changes-nothing",
		Claim: "Put leaves the store where the first write left it",
		Run:   repeatWriteChangesNothing,
		ProvenBy: batchedmixinstest.BrokenBatched(
			"a store where a rewrite is a second entry", planted(appendsEveryWrite),
		),
		ProvenReason: "unchanged",
	},

	{
		Method: "List", Name: "agrees-with-itself",
		Claim: "List agrees with itself",
		Run:   agreesWithItself,
		ProvenBy: batchedmixinstest.BrokenBatched(
			"a store that lists in a different order each time",
			planted(listsInVaryingOrder),
		),
		ProvenReason: "and the two agree",
	},

	{
		Method: "List", Name: "bounded-by-the-declared-limit",
		Claim: "List is bounded by the capacity the declaration gave it",
		Run:   boundedByTheDeclaredLimit,
		ProvenBy: batchedmixinstest.BrokenBatched(
			"a store that lists everything it holds", planted(ignoresTheLimit),
		),
		ProvenReason: "stops at the declared limit",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsBackWhatPutWrote(
	tb testing.TB, s batchedmixins.Batched, fx batchedmixinstest.BatchedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Value()), "the key is written")

	got, err := s.Read(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "the written key is found")
	testkit.Equal(tb, got, fx.Value(), "carrying what was written")
}

// repeatWriteChangesNothing is `idempotent`'s single-subject shadow: the row
// writes once, then repeats.
func repeatWriteChangesNothing(
	tb testing.TB, s batchedmixins.Batched, fx batchedmixinstest.BatchedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Value()), "the first write lands")

	before, err := s.List(tb.Context())
	testkit.NoError(tb, err, "the store can be listed")

	testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Value()), "the repeated write lands")

	after, err := s.List(tb.Context())
	testkit.NoError(tb, err, "and can still be listed")
	testkit.Equal(tb, after, before, "unchanged")
}

// agreesWithItself is `pure`'s shadow: the answer depends on the state and
// nothing else, and Go's map iteration is deliberately unordered — so a subject
// returning keys in range order fails here and passes everywhere a single call
// is compared against an expected set.
func agreesWithItself(
	tb testing.TB, s batchedmixins.Batched, fx batchedmixinstest.BatchedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Value()), "a key is written")
	testkit.NoError(tb, s.Put(tb.Context(), fx.KeyOther(), fx.ValueOther()),
		"and a second, so order is observable")

	first, err := s.List(tb.Context())
	testkit.NoError(tb, err, "the first listing succeeds")
	second, err := s.List(tb.Context())
	testkit.NoError(tb, err, "and so does the second")
	testkit.Equal(tb, second, first, "and the two agree")
}

// boundedByTheDeclaredLimit makes the number in the source answerable at all:
// `bounded limit=50` reaches the subject through the harness, so writing past
// it here is what puts the declaration under test.
func boundedByTheDeclaredLimit(
	tb testing.TB, s batchedmixins.Batched, _ batchedmixinstest.BatchedFixture,
) {
	tb.Helper()
	for i := range 60 {
		testkit.NoError(tb, s.Put(tb.Context(), fmt.Sprintf("k%02d", i), "v"), "a write lands")
	}
	got, err := s.List(tb.Context())
	testkit.NoError(tb, err, "the store can be listed")
	testkit.Len(tb, got, 50, "and the listing stops at the declared limit")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted store gets wrong, one per row.
//
// One implementation with a switch rather than four, because all four are the
// same store disagreeing about one of its three answers.
type fault int

const (
	// answersTheZero accepts the write, keeps it, and hands a reader the
	// zero value with no error — the shape a subject with a serialization
	// bug has, and the reason the row compares the value rather than the
	// error.
	answersTheZero fault = iota

	// appendsEveryWrite records a rewrite as a second entry, which is what
	// a store with no idempotence looks like from outside.
	appendsEveryWrite

	// listsInVaryingOrder answers a different order each call, which is
	// what listing straight out of a Go map does.
	listsInVaryingOrder

	// ignoresTheLimit lists everything it holds, whatever bound the
	// declaration gave it.
	ignoresTheLimit
)

// planted builds the constructor for one broken store. It takes the capacity
// the run hands every constructor, so the bound under test is the declared one
// rather than a number repeated here.
func planted(wrong fault) func(capacity int) *plantedBatched {
	return func(capacity int) *plantedBatched {
		return &plantedBatched{
			wrong: wrong, capacity: capacity, held: map[string]string{},
		}
	}
}

// plantedBatched keeps its keys in insertion order, which is stable — the one
// fault that varies it says so.
type plantedBatched struct {
	wrong    fault
	capacity int

	mu    sync.Mutex
	keys  []string
	held  map[string]string
	lists int
}

func (s *plantedBatched) Put(_ context.Context, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, seen := s.held[key]; !seen || s.wrong == appendsEveryWrite {
		s.keys = append(s.keys, key)
	}
	s.held[key] = value
	return nil
}

func (s *plantedBatched) Read(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.wrong == answersTheZero {
		return "", nil
	}
	return s.held[key], nil
}

func (s *plantedBatched) List(context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := slices.Clone(s.keys)
	if s.wrong == listsInVaryingOrder {
		s.lists++
		if s.lists%2 == 0 {
			slices.Reverse(out)
		}
	}
	if s.wrong == ignoresTheLimit {
		return out, nil
	}
	return out[:min(len(out), s.capacity)], nil
}
