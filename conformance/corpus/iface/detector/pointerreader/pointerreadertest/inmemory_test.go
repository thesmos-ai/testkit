// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The miss lives in nil, so the whole contract the signature can state is that
// dereferencing never has to happen: one generated check, about not crashing.
//
// This is the shape where the smoke check earns the most. A subject returning a
// pointer into its own map passes every generated check and hands the caller a
// handle on its state — which is the failure this fixture's own checks are
// about.
package pointerreadertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/pointerreader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/pointerreader/pointerreadertest"
)

// TestPointerReaderContract runs the generated check and this package's own.
func TestPointerReaderContract(t *testing.T) {
	t.Parallel()

	pointerreadertest.RunPointerReader(t, inMemory("in-memory"), pointerReaderChecks)
}

// TestPointerReaderContractWithoutSmoke drops a check through the typed index
// rather than a string, so a check that is renamed or stops being emitted
// breaks this compile instead of silently declining nothing.
func TestPointerReaderContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	pointerreadertest.RunPointerReader(t,
		inMemory("in-memory"),
		pointerreadertest.PointerReaderSuite.Without(
			pointerreadertest.PointerReaderSuite.Checks.Find.Smoke(),
		),
	)
}

// TestPointerReaderChecksCanFail drives every row against its planted defect.
func TestPointerReaderChecksCanFail(t *testing.T) {
	t.Parallel()

	pointerreadertest.ProvePointerReader(t, inMemory("in-memory"), pointerReaderChecks)
}

// --- Harnesses ---------------------------------------------------------------

// seededBody is what the seeded key holds, so a hit has something to carry
// beyond being non-nil.
const seededBody = "seeded"

func inMemory(name string) pointerreadertest.PointerReaderHarness[*pointerreadertest.InMemory] {
	return pointerreadertest.PointerReaderHarness[*pointerreadertest.InMemory]{
		Name: name, New: seeded,
	}
}

func seeded() *pointerreadertest.InMemory {
	s := pointerreadertest.NewInMemory()
	s.Put(pointerreader.Value{
		Key:  pointerreadertest.DefaultPointerReaderFixture().Key(),
		Body: seededBody,
	})
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------

var pointerReaderChecks = pointerreadertest.PointerReaderChecks{
	{
		Method: "Find", Name: "hit-returns-a-pointer",
		Claim: "Find returns a pointer to what was seeded",
		Run:   hitReturnsAPointer,
		ProvenBy: pointerreadertest.BrokenPointerReader(
			"a reader that answers nil for a key it holds", planted(missesItsOwnKey),
		),
		ProvenReason: "a present key reads as a pointer",
	},

	{
		Method: "Find", Name: "miss-reads-as-nil",
		Claim: "Find reads as nil for a key nothing wrote",
		Run:   missReadsAsNil,
		ProvenBy: pointerreadertest.BrokenPointerReader(
			"a reader that answers a pointer to nothing", planted(answersAZeroValue),
		),
		ProvenReason: "reads as nil rather than as a zero value",
	},
}

// --- Bodies -------------------------------------------------------------------

func hitReturnsAPointer(
	tb testing.TB, s pointerreader.PointerReader,
	fx pointerreadertest.PointerReaderFixture,
) {
	tb.Helper()
	got := s.Find(tb.Context(), fx.Key())
	testkit.True(tb, got != nil, "a present key reads as a pointer")
	testkit.Equal(tb, got.Body, seededBody, "to what was written")
}

// missReadsAsNil is the only signal this shape has. Nothing on the interface
// writes, so the generator refuses the miss and this is where it is stated.
func missReadsAsNil(
	tb testing.TB, s pointerreader.PointerReader,
	fx pointerreadertest.PointerReaderFixture,
) {
	tb.Helper()
	testkit.True(tb, s.Find(tb.Context(), fx.KeyOther()) == nil,
		"an absent key reads as nil rather than as a zero value")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted reader gets wrong about the nil.
//
// Both are the same confusion in opposite directions: nil is this shape's only
// way to say absent, so a reader that answers it for a hit loses data and one
// that withholds it for a miss invents some.
type fault int

const (
	// missesItsOwnKey answers nil for a key it holds, which a caller reads
	// as absence.
	missesItsOwnKey fault = iota

	// answersAZeroValue hands back a pointer to an empty value instead of
	// nil, which a caller cannot tell from a real record with empty fields.
	answersAZeroValue
)

// planted builds the constructor for one broken reader, holding what the
// harness's own subject holds so a hit has the same thing to hit.
func planted(wrong fault) func() plantedReader {
	return func() plantedReader {
		return plantedReader{
			wrong: wrong,
			key:   pointerreadertest.DefaultPointerReaderFixture().Key(),
		}
	}
}

type plantedReader struct {
	wrong fault
	key   string
}

func (p plantedReader) Find(_ context.Context, key string) *pointerreader.Value {
	if key != p.key {
		if p.wrong == answersAZeroValue {
			return &pointerreader.Value{}
		}
		return nil
	}
	if p.wrong == missesItsOwnKey {
		return nil
	}
	return &pointerreader.Value{Key: key, Body: seededBody}
}
