// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `stableorder` is the model tier's: that two drains agree is a claim about a
// pair of reads, and the derived reference is what compares them. What one
// subject settles is that the one order it answers is the declared one.
package stableordertest_test

import (
	"context"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/stableorder"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/stableorder/stableordertest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	stableordertest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	stableordertest.RunMixed(t,
		inMemory("in-memory"),
		stableordertest.MixedSuite.Without(stableordertest.MixedSuite.Checks.Add.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	stableordertest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) stableordertest.MixedHarness[*stableordertest.InMemory] {
	return stableordertest.MixedHarness[*stableordertest.InMemory]{
		Name: name, New: stableordertest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = stableordertest.MixedChecks{
	{
		Method: "Items", Name: "drains-in-key-order",
		Claim: "Items yields what Add put in, in key order",
		Run:   drainsInKeyOrder,
		ProvenBy: stableordertest.BrokenMixed(
			"a collection that drains its appends as they arrived", newKeepsArrivalOrder,
		),
		ProvenReason: "ordered rather than arbitrary",
	},
}

// --- Bodies -------------------------------------------------------------------

// drainsInKeyOrder adds the two keys out of order, because with one element the
// drain's ordering is unobservable and a subject returning map order would
// pass.
func drainsInKeyOrder(
	tb testing.TB, s stableorder.Mixed, _ stableordertest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Add(tb.Context(), stableorder.Value{Key: lateKey, Body: "last"}),
		"an element is accepted")
	testkit.NoError(tb, s.Add(tb.Context(), stableorder.Value{Key: earlyKey, Body: "first"}),
		"and a second that sorts ahead of it")

	got, err := s.Items(tb.Context())
	testkit.NoError(tb, err, "the drain succeeds")
	testkit.Equal(tb, len(got), 2, "each append is one element")
	testkit.Equal(tb, got[0].Key, earlyKey, "and the drain is ordered rather than arbitrary")
}

// --- Planted defects ----------------------------------------------------------

// The two keys the row adds, named so the ordering claim reads as one: earlyKey
// sorts ahead of lateKey, and the row adds them the other way round.
const (
	earlyKey = "aa"
	lateKey  = "zz"
)

// keepsArrivalOrder drains in the order things arrived, which agrees with
// itself on every read and is not the declared order — the reason this row is
// about the ORDER and the model tier's law is about two drains agreeing.
type keepsArrivalOrder struct{ items []stableorder.Value }

func newKeepsArrivalOrder() *keepsArrivalOrder { return &keepsArrivalOrder{} }

func (k *keepsArrivalOrder) Add(_ context.Context, v stableorder.Value) error {
	k.items = append(k.items, v)
	return nil
}

func (k *keepsArrivalOrder) Items(context.Context) ([]stableorder.Value, error) {
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

	stableordertest.MixedModelSaturation(t, func() stableordertest.Mixed { return stableordertest.NewInMemory() })
}
