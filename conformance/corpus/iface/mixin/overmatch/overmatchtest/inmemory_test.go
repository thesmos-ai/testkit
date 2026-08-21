// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `overmatch` is the model tier's: that a query answers no more than it should
// is a claim about a generated corpus and a generated query. What one subject
// settles is the row below — a repeated append is one element, in order.
package overmatchtest_test

import (
	"context"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/overmatch"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/overmatch/overmatchtest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	overmatchtest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	overmatchtest.RunMixed(t,
		inMemory("in-memory"),
		overmatchtest.MixedSuite.Without(overmatchtest.MixedSuite.Checks.Add.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	overmatchtest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) overmatchtest.MixedHarness[*overmatchtest.InMemory] {
	return overmatchtest.MixedHarness[*overmatchtest.InMemory]{
		Name: name, New: overmatchtest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = overmatchtest.MixedChecks{
	{
		Method: "Items", Name: "yields-each-append-once",
		Claim: "Items yields what Add put in, once each",
		Run:   yieldsEachAppendOnce,
		ProvenBy: overmatchtest.BrokenMixed(
			"a collection that drains its appends as they arrived", newKeepsTheRepeat,
		),
		ProvenReason: "the repeated key is one element",
	},
}

// --- Bodies -------------------------------------------------------------------

// yieldsEachAppendOnce puts the same key in twice and a second key that sorts
// ahead of it.
//
// The repeat is what makes the dedup claim testable: a drain that yielded it
// twice would be reporting its input rather than its contents. The out-of-order
// second key is what makes the ordering observable — with one element every
// order is the right one, and a subject returning map order would pass.
func yieldsEachAppendOnce(
	tb testing.TB, s overmatch.Mixed, _ overmatchtest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Add(tb.Context(), overmatch.Value{Key: lateKey, Body: "last"}),
		"an element is accepted")
	testkit.NoError(tb, s.Add(tb.Context(), overmatch.Value{Key: lateKey, Body: "last"}),
		"and so is the same key again")
	testkit.NoError(tb, s.Add(tb.Context(), overmatch.Value{Key: earlyKey, Body: "first"}),
		"and a second that sorts ahead of it")

	got, err := s.Items(tb.Context())
	testkit.NoError(tb, err, "the drain succeeds")
	testkit.Equal(tb, len(got), 2, "the repeated key is one element")
	testkit.Equal(tb, got[0].Key, earlyKey, "and the drain is ordered rather than arbitrary")
}

// --- Planted defects ----------------------------------------------------------

// The two keys the row adds, named so the ordering claim reads as one: earlyKey
// sorts ahead of lateKey, and the row adds them the other way round.
const (
	earlyKey = "aa"
	lateKey  = "zz"
)

// keepsTheRepeat drains every append in arrival order, which answers MORE than
// the collection holds — the overmatch this mixin is named for, in its smallest
// form.
type keepsTheRepeat struct{ items []overmatch.Value }

func newKeepsTheRepeat() *keepsTheRepeat { return &keepsTheRepeat{} }

func (k *keepsTheRepeat) Add(_ context.Context, v overmatch.Value) error {
	k.items = append(k.items, v)
	return nil
}

func (k *keepsTheRepeat) Items(context.Context) ([]overmatch.Value, error) {
	return slices.Clone(k.items), nil
}

// TestMixedLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestMixedLawsCanSaturate(t *testing.T) {
	t.Parallel()

	overmatchtest.MixedModelSaturation(t, func() overmatchtest.Mixed { return overmatchtest.NewInMemory() })
}
