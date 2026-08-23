// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// concurrentreaders is the model tier's: what it claims is that readers do not
// disturb each other, which needs several of them running at once and a
// reference to compare against. The row below is the deterministic half — that
// a single reader sees a single write at all.
package concurrentreaderstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/concurrentreaders"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/concurrentreaders/concurrentreaderstest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	concurrentreaderstest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	concurrentreaderstest.RunMixed(t,
		inMemory("in-memory"),
		concurrentreaderstest.MixedSuite.Without(concurrentreaderstest.MixedSuite.Checks.Get.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	concurrentreaderstest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) concurrentreaderstest.MixedHarness[*concurrentreaderstest.InMemory] {
	return concurrentreaderstest.MixedHarness[*concurrentreaderstest.InMemory]{
		Name: name, New: concurrentreaderstest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = concurrentreaderstest.MixedChecks{
	{
		Method: "Get", Name: "reads-back-what-put-wrote",
		Claim: "Get returns what Put wrote",
		Run:   readsBackWhatPutWrote,
		ProvenBy: concurrentreaderstest.BrokenMixed(
			"a store that answers the zero for a key it holds", newAnswersTheZero,
		),
		ProvenReason: "carries what was written",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsBackWhatPutWrote(
	tb testing.TB, s concurrentreaders.Mixed, fx concurrentreaderstest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Value()), "the key is written")

	got, err := s.Get(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a written key is found")
	testkit.Equal(tb, got, fx.Value(), "and carries what was written")
}

// --- Planted defects ----------------------------------------------------------

// answersTheZero keeps the write and hands a reader nothing, which is what a
// store serving reads from an index it never populated does — and which no
// check reading only the error can see.
type answersTheZero struct{ held map[string]string }

func newAnswersTheZero() *answersTheZero {
	return &answersTheZero{held: map[string]string{}}
}

func (a *answersTheZero) Put(_ context.Context, key, value string) error {
	a.held[key] = value
	return nil
}

func (a *answersTheZero) Get(_ context.Context, key string) (string, error) {
	if _, held := a.held[key]; !held {
		return "", concurrentreaderstest.ErrNotFound
	}
	return "", nil
}
