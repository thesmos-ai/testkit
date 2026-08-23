// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// idempotent is the model tier's: repeating a call leaves observable state
// unchanged, which needs a reference to observe against. What one subject and a
// fixed sequence settle is the row below — that a write is readable at all.
package idempotenttest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotent"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotent/idempotenttest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	idempotenttest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	idempotenttest.RunMixed(t,
		inMemory("in-memory"),
		idempotenttest.MixedSuite.Without(idempotenttest.MixedSuite.Checks.Put.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	idempotenttest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) idempotenttest.MixedHarness[*idempotenttest.InMemory] {
	return idempotenttest.MixedHarness[*idempotenttest.InMemory]{
		Name: name, New: idempotenttest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = idempotenttest.MixedChecks{
	{
		Method: "Read", Name: "reads-back-what-put-wrote",
		Claim: "Read returns what Put wrote",
		Run:   readsBackWhatPutWrote,
		ProvenBy: idempotenttest.BrokenMixed(
			"a store where a repeated write clears the key", newClearsOnRepeat,
		),
		ProvenReason: "carries what was written",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsBackWhatPutWrote(
	tb testing.TB, s idempotent.Mixed, fx idempotenttest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Value()), "the key is written")

	got, err := s.Read(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a written key is found")
	testkit.Equal(tb, got, fx.Value(), "and carries what was written")
}

// --- Planted defects ----------------------------------------------------------

// clearsOnRepeat takes the first write and empties the key on the second, which
// is what a store confusing "repeat changes nothing" with "repeat undoes it"
// does. The generated Put/idempotent check drives exactly one repeat, so this
// is the near miss the row beside it exists to catch.
type clearsOnRepeat struct{ seen map[string]int }

func newClearsOnRepeat() *clearsOnRepeat { return &clearsOnRepeat{seen: map[string]int{}} }

func (c *clearsOnRepeat) Put(_ context.Context, key, _ string) error {
	c.seen[key]++
	return nil
}

func (c *clearsOnRepeat) Read(_ context.Context, key string) (string, error) {
	if c.seen[key] == 0 {
		return "", idempotenttest.ErrNotFound
	}
	return "", nil
}
