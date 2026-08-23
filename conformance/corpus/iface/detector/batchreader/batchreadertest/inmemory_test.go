// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// A variadic read witnesses one key per generated check, which is the narrowing
// the generated check type and fixture field both state.
//
// So everything a *batch* read is for — order, arity, the empty call, all-or-
// nothing on a partial miss — is written here. None of it is a property one
// derived value can reach, and the fixture holds one value per parameter.
package batchreadertest_test

import (
	"context"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/batchreader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/batchreader/batchreadertest"
)

// TestBatchReaderContract runs the generated checks and this package's own.
func TestBatchReaderContract(t *testing.T) {
	t.Parallel()

	batchreadertest.RunBatchReader(t, inMemory("in-memory"), batchReaderChecks)
}

// TestBatchReaderContractWithoutSmoke drops a check through the typed index
// rather than a string, so a check that is renamed or stops being emitted
// breaks this compile instead of silently declining nothing.
func TestBatchReaderContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	batchreadertest.RunBatchReader(t,
		inMemory("in-memory"),
		batchreadertest.BatchReaderSuite.Without(
			batchreadertest.BatchReaderSuite.Checks.GetAll.Smoke(),
		),
	)
}

// TestBatchReaderChecksCanFail drives every row against its planted defect.
func TestBatchReaderChecksCanFail(t *testing.T) {
	t.Parallel()

	batchreadertest.ProveBatchReader(t, inMemory("in-memory"), batchReaderChecks)
}

// TestBatchReaderAnswersPerKeyWhenItHoldsThemAll runs a subject holding BOTH
// derived keys, which the run above cannot be in: the row that asks for a
// partial miss needs KeysOther absent.
func TestBatchReaderAnswersPerKeyWhenItHoldsThemAll(t *testing.T) {
	t.Parallel()

	batchreadertest.RunBatchReader(t, holdingBothKeys("in-memory, holding both keys"))
}

// --- Harnesses ---------------------------------------------------------------

// secondKey is a present key this package names for itself.
//
// The fixture's Keys and KeysOther are a hit and a miss, and the rows below
// depend on that: taking KeysOther for a second present key would make the
// miss row assert nothing.
const secondKey = "second-key"

// absentKey is held by no subject here, so a batch naming it has a real absence
// to fail on.
const absentKey = "held-by-nobody"

// The bodies the two seeded keys carry, so an order claim compares values
// rather than positions.
const (
	firstBody  = "first"
	secondBody = "second"
)

// inMemory seeds Keys and deliberately not KeysOther, so the all-or-nothing row
// has a real absence to fail on.
func inMemory(name string) batchreadertest.BatchReaderHarness[*batchreadertest.InMemory] {
	return batchreadertest.BatchReaderHarness[*batchreadertest.InMemory]{Name: name, New: seeded}
}

func seeded() *batchreadertest.InMemory {
	fx := batchreadertest.DefaultBatchReaderFixture()
	s := batchreadertest.NewInMemory()
	s.Put(batchreader.Value{Key: fx.Keys(), Body: firstBody})
	s.Put(batchreader.Value{Key: secondKey, Body: secondBody})
	return s
}

// holdingBothKeys seeds both DERIVED keys, which is the state the generated
// per-key checks want and the partial-miss row forbids.
func holdingBothKeys(
	name string,
) batchreadertest.BatchReaderHarness[*batchreadertest.InMemory] {
	return batchreadertest.BatchReaderHarness[*batchreadertest.InMemory]{
		Name: name,
		New: func() *batchreadertest.InMemory {
			fx := batchreadertest.DefaultBatchReaderFixture()
			s := batchreadertest.NewInMemory()
			s.Put(batchreader.Value{Key: fx.Keys()})
			s.Put(batchreader.Value{Key: fx.KeysOther()})
			return s
		},
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var batchReaderChecks = batchreadertest.BatchReaderChecks{
	{
		Method: "GetAll", Name: "answers-in-the-order-asked",
		Claim: "GetAll answers several keys in the order asked",
		Run:   answersInTheOrderAsked,
		ProvenBy: batchreadertest.BrokenBatchReader(
			"a reader that answers in the order it happens to hold",
			planted(answersInStoreOrder),
		),
		ProvenReason: "follows the question rather than the store",
	},

	{
		Method: "GetAll", Name: "nothing-rather-than-a-partial-answer",
		Claim: "GetAll returns nothing rather than a partial answer",
		Run:   nothingRatherThanAPartialAnswer,
		ProvenBy: batchreadertest.BrokenBatchReader(
			"a reader that drops the keys it could not find",
			planted(answersPartially),
		),
		ProvenReason: "one absent key fails the batch",
	},

	{
		Method: "GetAll", Name: "empty-call-succeeds",
		Claim: "GetAll succeeds on the empty call",
		Run:   emptyCallSucceeds,
		ProvenBy: batchreadertest.BrokenBatchReader(
			"a reader that reads no keys as a missing key",
			planted(refusesTheEmptyCall),
		),
		ProvenReason: "asking for nothing is not a failure",
	},
}

// --- Bodies -------------------------------------------------------------------

func answersInTheOrderAsked(
	tb testing.TB, s batchreader.BatchReader, fx batchreadertest.BatchReaderFixture,
) {
	tb.Helper()
	got, err := s.GetAll(tb.Context(), fx.Keys(), secondKey)
	testkit.NoError(tb, err, "a batch of held keys succeeds")
	testkit.Equal(tb, got, []batchreader.Value{
		{Key: fx.Keys(), Body: firstBody},
		{Key: secondKey, Body: secondBody},
	}, "and comes back in the order it was asked for")

	reversed, err := s.GetAll(tb.Context(), secondKey, fx.Keys())
	testkit.NoError(tb, err, "so does the same batch reversed")
	testkit.Equal(tb, reversed[0].Key, secondKey,
		"and the answer follows the question rather than the store")
}

// nothingRatherThanAPartialAnswer is the failure mode a batch read has and a
// single read does not: a caller cannot tell a short result from a complete one
// without comparing lengths, so dropping the absent key silently is worse than
// failing.
func nothingRatherThanAPartialAnswer(
	tb testing.TB, s batchreader.BatchReader, fx batchreadertest.BatchReaderFixture,
) {
	tb.Helper()
	got, err := s.GetAll(tb.Context(), fx.Keys(), absentKey)
	testkit.ErrorIs(tb, err, batchreadertest.ErrNotFound, "one absent key fails the batch")
	testkit.True(tb, got == nil, "and nothing is returned beside the error")
}

// emptyCallSucceeds is the call no derivation reaches: a fixture holds one
// value per parameter, so a generated check always passes exactly one.
func emptyCallSucceeds(
	tb testing.TB, s batchreader.BatchReader, _ batchreadertest.BatchReaderFixture,
) {
	tb.Helper()
	got, err := s.GetAll(tb.Context())
	testkit.NoError(tb, err, "asking for nothing is not a failure")
	testkit.Len(tb, got, 0, "and answers with nothing")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted reader gets wrong about a BATCH. None of the
// three is wrong about a single key, which is why no generated check catches
// them and every one of them needs a row.
type fault int

const (
	// answersInStoreOrder hands back what it holds in its own order, which
	// a caller matching answers to questions by position reads as a
	// mislabelled result.
	answersInStoreOrder fault = iota

	// answersPartially drops the keys it could not find and reports
	// success, so a short result is indistinguishable from a complete one.
	answersPartially

	// refusesTheEmptyCall reads a call with no keys as a key it could not
	// find.
	refusesTheEmptyCall
)

// planted builds the constructor for one broken reader, holding what the
// harness's own subject holds so a batch has the same thing to find.
func planted(wrong fault) func() *plantedReader {
	return func() *plantedReader {
		fx := batchreadertest.DefaultBatchReaderFixture()
		return &plantedReader{
			wrong: wrong,
			order: []string{fx.Keys(), secondKey},
			held: map[string]batchreader.Value{
				fx.Keys(): {Key: fx.Keys(), Body: firstBody},
				secondKey: {Key: secondKey, Body: secondBody},
			},
		}
	}
}

// plantedReader keeps an insertion order beside its map, because "the order it
// happens to hold" has to be a stable wrong answer rather than a random one: a
// defect that sometimes agrees with the question is a proof that sometimes
// fails.
type plantedReader struct {
	wrong fault
	order []string
	held  map[string]batchreader.Value
}

func (r *plantedReader) GetAll(
	_ context.Context, keys ...string,
) ([]batchreader.Value, error) {
	if len(keys) == 0 && r.wrong == refusesTheEmptyCall {
		return nil, batchreadertest.ErrNotFound
	}
	asked := keys
	if r.wrong == answersInStoreOrder {
		asked = r.inStoreOrder(keys)
	}
	out := make([]batchreader.Value, 0, len(asked))
	for _, k := range asked {
		v, held := r.held[k]
		if !held {
			if r.wrong == answersPartially {
				continue
			}
			return nil, batchreadertest.ErrNotFound
		}
		out = append(out, v)
	}
	return out, nil
}

// inStoreOrder re-sorts the question into the order this reader holds things.
func (r *plantedReader) inStoreOrder(keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, k := range r.order {
		if slices.Contains(keys, k) {
			out = append(out, k)
		}
	}
	for _, k := range keys {
		if !slices.Contains(r.order, k) {
			out = append(out, k)
		}
	}
	return out
}
