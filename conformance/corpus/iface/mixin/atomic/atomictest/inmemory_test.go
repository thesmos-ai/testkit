// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// atomic is the model tier's under ADR-0028 — AUTO-ATOMIC-WRITE states it,
// comparing observable state around the write the subject refuses.
//
// The rows below are the deterministic complement: one accepted entry read back
// whole, and one refused entry that landed nowhere.
package atomictest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/atomic"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/atomic/atomictest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	atomictest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	atomictest.RunMixed(t,
		inMemory("in-memory"),
		atomictest.MixedSuite.Without(atomictest.MixedSuite.Checks.Write.Smoke()),
	)
}

// TestMixedChecksCanFail drives every row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	atomictest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

// The two halves a whole entry carries. Both non-empty, because the claim is
// that an entry lands entire — one empty half would be indistinguishable from
// the half a broken subject dropped.
const (
	leftHalf  = "left"
	rightHalf = "right"
)

func inMemory(name string) atomictest.MixedHarness[*atomictest.InMemory] {
	return atomictest.MixedHarness[*atomictest.InMemory]{
		Name: name, New: atomictest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = atomictest.MixedChecks{
	{
		Method: "Read", Name: "returns-the-whole-entry",
		Claim: "Read returns the whole entry as it was written",
		Run:   returnsTheWholeEntry,
		ProvenBy: atomictest.BrokenMixed(
			"a store that keeps one half of what it was given", planted(keepsOneHalf),
		),
		ProvenReason: "carries both halves",
	},

	{
		Method: "Read", Name: "half-an-entry-lands-nowhere",
		Claim: "Read refuses half an entry whole",
		Run:   halfAnEntryLandsNowhere,
		ProvenBy: atomictest.BrokenMixed(
			"a store that reports the refusal and writes anyway",
			planted(keepsWhatItRefused),
		),
		ProvenReason: "and nothing landed",
	},
}

// --- Bodies -------------------------------------------------------------------

func returnsTheWholeEntry(tb testing.TB, s atomic.Mixed, fx atomictest.MixedFixture) {
	tb.Helper()
	e := atomic.Entry{Key: fx.Key(), Left: leftHalf, Right: rightHalf}
	testkit.NoError(tb, s.Write(tb.Context(), e), "a whole entry lands")

	got, err := s.Read(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a written key is found")
	testkit.Equal(tb, got, e, "and carries both halves")
}

// halfAnEntryLandsNowhere is the refusal's other half: reporting the error is
// not enough if the entry landed anyway, which is what "atomic" is about.
func halfAnEntryLandsNowhere(tb testing.TB, s atomic.Mixed, fx atomictest.MixedFixture) {
	tb.Helper()
	half := atomic.Entry{Key: fx.KeyOther(), Left: leftHalf}
	testkit.ErrorIs(tb, s.Write(tb.Context(), half), atomictest.ErrHalfEntry,
		"an entry missing one half is refused")

	_, err := s.Read(tb.Context(), fx.KeyOther())
	testkit.ErrorIs(tb, err, atomictest.ErrNotFound, "and nothing landed")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted store gets wrong.
type fault int

const (
	// keepsOneHalf writes the left and drops the right, which is the torn
	// write the mixin is named for.
	keepsOneHalf fault = iota

	// keepsWhatItRefused reports the refusal and writes anyway, which is
	// the shape a store validating after the write has — and the one a
	// check reading only the error would call correct.
	keepsWhatItRefused
)

// planted builds the constructor for one broken store.
func planted(wrong fault) func() *plantedStore {
	return func() *plantedStore {
		return &plantedStore{wrong: wrong, held: map[string]atomic.Entry{}}
	}
}

type plantedStore struct {
	wrong fault
	held  map[string]atomic.Entry
}

func (p *plantedStore) Write(_ context.Context, e atomic.Entry) error {
	if e.Left == "" || e.Right == "" {
		if p.wrong == keepsWhatItRefused {
			p.held[e.Key] = e
		}
		return atomictest.ErrHalfEntry
	}
	if p.wrong == keepsOneHalf {
		e.Right = ""
	}
	p.held[e.Key] = e
	return nil
}

func (p *plantedStore) Read(_ context.Context, key string) (atomic.Entry, error) {
	e, held := p.held[key]
	if !held {
		return atomic.Entry{}, atomictest.ErrNotFound
	}
	return e, nil
}
