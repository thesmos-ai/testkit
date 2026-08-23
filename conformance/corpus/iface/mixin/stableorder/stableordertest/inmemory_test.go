// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `stableorder` is the model tier's: that two drains agree is a claim about a
// pair of reads, and the derived reference is what compares them. What one
// subject settles is that the one order it answers is the declared one.
package stableordertest_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/stableorder"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/stableorder/stableordertest"
	"go.thesmos.sh/testkit/engine/legs"
	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/action"
	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/engine/suite"
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

	stableordertest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) stableordertest.MixedHarness[*stableordertest.InMemory] {
	return stableordertest.MixedHarness[*stableordertest.InMemory]{
		Name: name, New: stableordertest.NewInMemory,
		// AUTO-STREAM-STABLE-ORDER needs the order the drain claims to be
		// in, and which field carries it is a fact about Value rather than
		// about its shape. Key ascending is what Items sorts on.
		Provide: map[suite.Capability]any{
			"less": func(a, b stableorder.Value) bool { return a.Key < b.Key },
		},
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
	{
		Name:    "every-add-is-observable",
		Claim:   "after every Add, Items answers at least what the run has put in",
		RunWith: everyAddIsObservable,
		ProvenBy: stableordertest.BrokenMixed(
			"a collection that accepts an append and keeps nothing", newDropsEveryAdd,
		),
		ProvenReason: "answered nothing after an Add it accepted",
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

// everyAddIsObservable states a claim over a SEQUENCE rather than a fixed
// pair of calls, which is what law.AfterEvery is for: the predicate runs
// after each occurrence of the named action, however many the run draws
// and whatever it interleaves between them.
//
// Written by hand because the combinator is a consumer's to reach for. It
// carries no law identifier — its ID is composed from the action it
// watches — so no selection rule can name it and no census over
// identifiers sees it. This is what exercises it.
//
// The predicate asks the weakest thing that a dropped append breaks: an
// append the subject ACCEPTED must leave the drain non-empty. Comparing
// counts against the reference would be the differential's claim, which
// the row above already carries.
func everyAddIsObservable(
	tb testing.TB, sub stableordertest.MixedSubject, fx stableordertest.MixedFixture,
) {
	tb.Helper()
	legs.Law(tb, sub,
		func() stableorder.Mixed { return sub.New(tb) },
		stableordertest.NewMixedModelReference,
		[]model.Action[stableorder.Mixed]{
			action.Writer("Add",
				model.SampledFrom([]stableorder.Value{fx.Value(), fx.ValueOther()}),
				func(ctx context.Context, s stableorder.Mixed, v stableorder.Value) error {
					return s.Add(ctx, v)
				}),
		},
		[]law.Law[stableorder.Mixed]{
			&law.AfterEvery[stableorder.Mixed]{
				ActionName: "Add",
				Predicate: func(rt *model.T, sut, _ stableorder.Mixed) error {
					items, err := sut.Items(rt.Context())
					if err != nil {
						return fmt.Errorf("the drain refused after an accepted Add: %w", err)
					}
					if len(items) == 0 {
						return errors.New("the drain answered nothing after an Add it accepted")
					}
					return nil
				},
			},
		})
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

// dropsEveryAdd accepts every append and keeps none, which is invisible to
// a check that adds and drains in one breath only because that check adds
// first — this one is caught by the predicate that runs after each append
// the run draws.
type dropsEveryAdd struct{}

func newDropsEveryAdd() *dropsEveryAdd { return &dropsEveryAdd{} }

func (*dropsEveryAdd) Add(context.Context, stableorder.Value) error { return nil }

func (*dropsEveryAdd) Items(context.Context) ([]stableorder.Value, error) { return nil, nil }
