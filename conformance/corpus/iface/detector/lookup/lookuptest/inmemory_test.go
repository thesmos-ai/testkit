// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// No context and no error leaves exactly one generated check: the smoke call.
//
// That is the floor, and it is still worth having — a method that panics on a
// derived key is one nothing else in the file would reach. Everything else the
// shape means is written here: nothing on this interface writes, so the miss
// the header refuses is one only a seeding consumer can state.
package lookuptest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/lookup"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/lookup/lookuptest"
)

// TestLookupContract runs the generated check and this package's own.
func TestLookupContract(t *testing.T) {
	t.Parallel()

	lookuptest.RunLookup(t, inMemory("in-memory"), lookupChecks)
}

// TestLookupContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestLookupContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	lookuptest.RunLookup(t,
		inMemory("in-memory"),
		lookuptest.LookupSuite.Without(lookuptest.LookupSuite.Checks.Inspect.Smoke()),
	)
}

// TestLookupChecksCanFail drives every row against its planted defect.
func TestLookupChecksCanFail(t *testing.T) {
	t.Parallel()

	lookuptest.ProveLookup(t, inMemory("in-memory"), lookupChecks)
}

// --- Harnesses ---------------------------------------------------------------

// What the seeded key holds in each of its two slots. Both non-zero, because
// every claim below is that the slots agree — and a zero in either would make
// the hit row satisfiable by a subject that answers nothing.
const (
	seededBody     = "seeded"
	seededRevision = 3
)

func inMemory(name string) lookuptest.LookupHarness[*lookuptest.InMemory] {
	return lookuptest.LookupHarness[*lookuptest.InMemory]{Name: name, New: seeded}
}

func seeded() *lookuptest.InMemory {
	s := lookuptest.NewInMemory()
	s.Put(
		lookup.Value{Key: lookuptest.DefaultLookupFixture().Key(), Body: seededBody},
		lookup.Meta{Revision: seededRevision},
	)
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------
//
// Two slots and a flag, so every planted defect gets ONE of the three right and
// the rest wrong — which is the failure this shape has and a single-value read
// does not.

var lookupChecks = lookuptest.LookupChecks{
	{
		Method: "Inspect", Name: "both-slots-for-a-hit",
		Claim: "Inspect returns both slots for a hit",
		Run:   bothSlotsForAHit,
		ProvenBy: lookuptest.BrokenLookup(
			"a store whose metadata slot stays zero on a hit",
			planted(forgetsTheMetadata),
		),
		ProvenReason: "the metadata slot agrees",
	},

	{
		Method: "Inspect", Name: "both-slots-zero-on-a-miss",
		Claim: "Inspect zeroes both slots for a key nothing wrote",
		Run:   bothSlotsZeroOnAMiss,
		ProvenBy: lookuptest.BrokenLookup(
			"a store that zeroes the value and leaves the metadata behind",
			planted(leaksTheMetadata),
		),
		ProvenReason: "and so is the metadata slot",
	},
}

// --- Bodies -------------------------------------------------------------------

func bothSlotsForAHit(tb testing.TB, s lookup.Lookup, fx lookuptest.LookupFixture) {
	tb.Helper()
	v, m, ok := s.Inspect(fx.Key())
	testkit.True(tb, ok, "a seeded key is reported present")
	testkit.Equal(tb, v.Body, seededBody, "the value slot carries what was written")
	testkit.Equal(tb, m.Revision, seededRevision, "and the metadata slot agrees")
}

// bothSlotsZeroOnAMiss asks about both, not one: a subject zeroing the value
// and leaking the metadata satisfies a check that reads one slot of two.
func bothSlotsZeroOnAMiss(tb testing.TB, s lookup.Lookup, fx lookuptest.LookupFixture) {
	tb.Helper()
	v, m, ok := s.Inspect(fx.KeyOther())
	testkit.False(tb, ok, "an unwritten key is reported absent")
	testkit.Equal(tb, v, lookup.Value{}, "the value slot is the zero")
	testkit.Equal(tb, m, lookup.Meta{}, "and so is the metadata slot")
}

// --- Planted defects ----------------------------------------------------------

// fault names which slot one planted store gets wrong.
type fault int

const (
	// forgetsTheMetadata answers the value and leaves the metadata at its
	// zero, which is a store reading one column of two.
	forgetsTheMetadata fault = iota

	// leaksTheMetadata clears the value on a miss and hands back whatever
	// metadata was in the buffer, which is the half-cleared answer the
	// second row exists to forbid.
	leaksTheMetadata
)

// planted builds the constructor for one broken store, holding what the
// harness's own subject holds so a hit has the same thing to hit.
func planted(wrong fault) func() plantedLookup {
	return func() plantedLookup {
		return plantedLookup{wrong: wrong, key: lookuptest.DefaultLookupFixture().Key()}
	}
}

type plantedLookup struct {
	wrong fault
	key   string
}

func (p plantedLookup) Inspect(key string) (lookup.Value, lookup.Meta, bool) {
	if key != p.key {
		if p.wrong == leaksTheMetadata {
			return lookup.Value{}, lookup.Meta{Revision: seededRevision}, false
		}
		return lookup.Value{}, lookup.Meta{}, false
	}
	if p.wrong == forgetsTheMetadata {
		return lookup.Value{Key: key, Body: seededBody}, lookup.Meta{}, true
	}
	return lookup.Value{Key: key, Body: seededBody}, lookup.Meta{Revision: seededRevision}, true
}
