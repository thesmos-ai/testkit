// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The value the row stores is built here rather than drawn from the fixture,
// because the fixture's is nil: eidos writes no literal for a pointer, so the
// field is declared and left at its zero.
//
// Which means for THIS signature the generated nilsafe check and the smoke call
// make the same call. The mixin earns its place on a method whose parameters
// are derivable — a struct, a map, a slice — where smoke passes a real value
// and only the nilsafe check passes zeros.
package nilsafetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/nilsafe"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/nilsafe/nilsafetest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	nilsafetest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	nilsafetest.RunMixed(t,
		inMemory("in-memory"),
		nilsafetest.MixedSuite.Without(nilsafetest.MixedSuite.Checks.Store.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	nilsafetest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) nilsafetest.MixedHarness[*nilsafetest.InMemory] {
	return nilsafetest.MixedHarness[*nilsafetest.InMemory]{
		Name: name, New: nilsafetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = nilsafetest.MixedChecks{
	{
		Method: "Store", Name: "accepts-a-payload-it-was-given",
		Claim: "Store stores a payload it was given",
		Run:   acceptsAPayloadItWasGiven,
		ProvenBy: nilsafetest.BrokenMixed(
			"a store that refuses every payload it is handed", newRefusesEverything,
		),
		ProvenReason: "a non-nil payload is accepted",
	},
}

// --- Bodies -------------------------------------------------------------------

func acceptsAPayloadItWasGiven(
	tb testing.TB, s nilsafe.Mixed, _ nilsafetest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Store(tb.Context(), &nilsafe.Payload{Key: "k", Body: "b"}),
		"a non-nil payload is accepted")
}

// --- Planted defects ----------------------------------------------------------

// refusesEverything answers the nil sentinel for a payload that is plainly
// there, which is the guard written against the wrong condition — and the
// mirror of what the generated nilsafe check catches.
type refusesEverything struct{}

func newRefusesEverything() refusesEverything { return refusesEverything{} }

func (refusesEverything) Store(context.Context, *nilsafe.Payload) error {
	return nilsafetest.ErrNilPayload
}
