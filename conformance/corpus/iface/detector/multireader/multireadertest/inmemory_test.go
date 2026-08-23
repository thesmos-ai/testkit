// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// A read returning value and metadata owes the zero for both, not for the
// first.
//
// That is the whole difference from the plain reader shape, and it is invisible
// in a harness that compiles: a check comparing one slot renders identically to
// one comparing two. Only a subject that zeroes some and not others tells them
// apart, which is what TestGetWithMetaZeroesEverySlot is for.
package multireadertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/multireader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/multireader/multireadertest"
)

// TestMultiReaderContract runs the generated checks and this package's own.
func TestMultiReaderContract(t *testing.T) {
	t.Parallel()

	multireadertest.RunMultiReader(t, inMemory("in-memory"), multiReaderChecks)
}

// TestMultiReaderContractWithoutSmoke drops a check through the typed index
// rather than a string, so a check that is renamed or stops being emitted
// breaks this compile instead of silently declining nothing.
func TestMultiReaderContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	multireadertest.RunMultiReader(t,
		inMemory("in-memory"),
		multireadertest.MultiReaderSuite.Without(
			multireadertest.MultiReaderSuite.Checks.GetWithMeta.Smoke(),
		),
	)
}

// TestMultiReaderChecksCanFail drives the row against its planted defect.
func TestMultiReaderChecksCanFail(t *testing.T) {
	t.Parallel()

	multireadertest.ProveMultiReader(t, inMemory("in-memory"), multiReaderChecks)
}

// TestGetWithMetaZeroesEverySlot pins the GENERATED check rather than the row:
// a subject zeroing only its first slot must fail, or the check reads one of
// two and reports on both.
//
// The check is reached as data rather than by name: the assertion functions are
// unexported, and Suite is the seam a runner of your own would use.
func TestGetWithMetaZeroesEverySlot(t *testing.T) {
	t.Parallel()

	fx := multireadertest.DefaultMultiReaderFixture()
	want := multireadertest.MultiReaderSuite.Checks.GetWithMeta.ZeroOnError()

	var zeroOnError func(tb testing.TB, s multireader.MultiReader)
	for _, c := range multireadertest.MultiReaderSuite.Suite(fx).Checks {
		if c.ID == want {
			zeroOnError = c.Run
		}
	}
	testkit.True(t, zeroOnError != nil, "the run emits the check this test is about")

	f := testkit.NewFailableTB()
	zeroOnError(f, partialZero{})

	testkit.True(t, f.Failed(),
		"metadata leaked beside an error must fail, whichever slot it is")
}

// --- Harnesses ---------------------------------------------------------------

// What the seeded key holds in each of its two slots.
const (
	seededBody     = "seeded"
	seededRevision = 7
)

// inMemory seeds the subject: MultiReader declares no writer, so the hit path
// is unreachable without a seeded constructor.
func inMemory(name string) multireadertest.MultiReaderHarness[*multireadertest.InMemory] {
	return multireadertest.MultiReaderHarness[*multireadertest.InMemory]{
		Name: name, New: seeded,
	}
}

func seeded() *multireadertest.InMemory {
	s := multireadertest.NewInMemory()
	s.Put(
		multireader.Value{Key: multireadertest.DefaultMultiReaderFixture().Key(), Body: seededBody},
		multireader.Meta{Revision: seededRevision},
	)
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------

var multiReaderChecks = multireadertest.MultiReaderChecks{
	{
		Method: "GetWithMeta", Name: "both-slots-for-a-hit",
		Claim: "GetWithMeta returns both slots for a hit",
		Run:   bothSlotsForAHit,
		ProvenBy: multireadertest.BrokenMultiReader(
			"a reader whose metadata slot stays zero on a hit", newForgetsTheMetadata,
		),
		ProvenReason: "the metadata slot agrees",
	},
}

// --- Bodies -------------------------------------------------------------------

func bothSlotsForAHit(
	tb testing.TB, s multireader.MultiReader, fx multireadertest.MultiReaderFixture,
) {
	tb.Helper()
	v, m, err := s.GetWithMeta(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a seeded key is found")
	testkit.Equal(tb, v.Body, seededBody, "the value slot carries what was written")
	testkit.Equal(tb, m.Revision, seededRevision, "and the metadata slot agrees")
}

// --- Planted defects ----------------------------------------------------------

// forgetsTheMetadata answers the value and leaves the metadata at its zero,
// which is the hit-path mirror of what partialZero does on a miss.
type forgetsTheMetadata struct{}

func newForgetsTheMetadata() forgetsTheMetadata { return forgetsTheMetadata{} }

func (forgetsTheMetadata) GetWithMeta(
	_ context.Context, key string,
) (multireader.Value, multireader.Meta, error) {
	return multireader.Value{Key: key, Body: seededBody}, multireader.Meta{}, nil
}

// partialZero reports a miss and leaks its metadata, which is the one violation
// a single-slot check cannot see. It is driven against the GENERATED check
// rather than named by a row, which is why it is not a ProvenBy.
type partialZero struct{}

func (partialZero) GetWithMeta(
	context.Context, string,
) (multireader.Value, multireader.Meta, error) {
	return multireader.Value{}, multireader.Meta{Revision: 9}, multireader.ErrNotFound
}
