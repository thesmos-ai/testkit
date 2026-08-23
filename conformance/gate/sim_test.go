// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
)

// crashClaim is the header line the crash refusal is filed under, and
// the same string the model tier writes it with.
const crashClaim = "crash recovery"

// TestACrashPairEitherRendersOrSaysWhy holds the refusal to the one
// thing that makes it honest: a reader can see it.
//
// A pair derives from almost any keyed reader and writer, and only some
// of those mean what the schedule holds a subject to — "an acknowledged
// write sits at its key until something overwrites it". A store that
// pins, deduplicates, stamps what it stores or answers reads at a
// timestamp all break that, and declining the row is correct for every
// one of them.
//
// What is not correct is declining it silently. No row is planned, so no
// manifest row goes missing and no check reports; the claim simply is
// not made, and a reader has no way to tell that from a claim nobody
// thought of. This is the direction that goes wrong quietly, which is
// why it is the one gated.
func TestACrashPairEitherRendersOrSaysWhy(t *testing.T) {
	t.Parallel()

	census, err := censusOnce()
	testkit.NoError(t, err, "the emission census runs")

	var silent []string
	for _, e := range census.Emitted {
		if !e.SimPair || e.Recovery {
			continue
		}
		if why := e.Refusals[crashClaim]; strings.TrimSpace(why) == "" {
			silent = append(silent, e.Dir)
		}
	}
	slices.Sort(silent)
	testkit.Len(t, silent, 0, "gives the crash schedule a pair to drive, states no claim "+
		"over it, and tells a reader nothing: "+strings.Join(silent, ", "))
}

// TestTheRecoveryLegStillRenders is the floor under the crash claim.
//
// A count rather than a per-fixture rule, because whether a given
// interface's acknowledgement survives a rebuild is a fact about that
// interface. What a floor catches is the leg quietly ceasing to derive
// at all — a refusal arm widened by one word, and eight fixtures stop
// stating a claim with every gate still green.
func TestTheRecoveryLegStillRenders(t *testing.T) {
	t.Parallel()

	census, err := censusOnce()
	testkit.NoError(t, err, "the emission census runs")

	var rendered []string
	for _, e := range census.Emitted {
		if e.Recovery {
			rendered = append(rendered, e.Dir)
		}
	}
	slices.Sort(rendered)
	testkit.True(t, len(rendered) >= recoveryFloor,
		"fewer fixtures state the crash claim than did; the leg derives on "+
			strings.Join(rendered, ", "))
}

// recoveryFloor is how many corpus fixtures render the crash-recovery
// leg.
//
// A floor and not an equality: a new fixture whose shape carries the
// claim raises it and nothing should have to be edited for that. It
// moves down only with a reason written beside it.
const recoveryFloor = 8

// TestEveryModelPackageOffersASugaredProperty counts the drawn-input
// fields a consumer is offered, one per method the tier can already draw
// an argument for.
//
// The un-sugared Prop is offered unconditionally and is not what this
// measures. The sugars are: they are the difference between writing a
// property and first reading the generated file to work out which pool
// your method's argument comes from, and they derive from facts — the
// shared pools, the actions — that can quietly stop holding.
func TestEveryModelPackageOffersASugaredProperty(t *testing.T) {
	t.Parallel()

	census, err := censusOnce()
	testkit.NoError(t, err, "the emission census runs")

	sugared := 0
	for _, e := range census.Emitted {
		if len(e.PropSugars) > 0 {
			sugared++
		}
	}
	testkit.True(t, sugared >= propSugarFloor,
		"the corpus offers sugared property fields on fewer fixtures than it did; "+
			"a surface that got smaller with nothing to show for it is what this catches")
}

// propSugarFloor is how many corpus fixtures offer at least one sugared
// property field — 62 of the 81 this tier emits for. A floor, on the
// same terms as [recoveryFloor].
//
// The nineteen without one are not a gap. A method whose argument comes
// from neither shared pool has no sugar to offer, and the un-sugared
// Prop covers it with the fixture in hand.
const propSugarFloor = 62
