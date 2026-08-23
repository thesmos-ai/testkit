// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
)

// TestEveryConcurrencyFamilyIsRenderedOrRegistered is the linearizability
// census: a fixture whose shape selects a concurrency model owes a leg that
// drives it, or a row saying why it has none.
//
// The derivation runs today and nothing renders it. Five families are
// classified — kv, lease, cas, append, session — the reader and writer
// actions are resolved, the entry type and the lease's two sentinels are
// carried, and every one of those facts is computed and dropped. A
// derivation with no output is the most expensive kind of dead code: it
// reads as a feature in the source and buys nothing in the emission, and
// nothing before this said so out loud.
//
// It is a register rather than a red because the leg is a feature, not a
// fix — a Porcupine model that steps an operation wrongly reports a
// linearizable history as a violation, or worse the reverse, and that is
// worse than the absence. The rows are what stop the absence being
// invisible while it is written.
func TestEveryConcurrencyFamilyIsRenderedOrRegistered(t *testing.T) {
	t.Parallel()

	census, err := censusOnce()
	testkit.NoError(t, err, "the emission census runs")

	var unrendered []string
	for _, e := range census.Emitted {
		if e.ConcFamily == "" {
			continue
		}
		if _, registered := gate.UnrenderedLegs[e.ConcFamily]; !e.Linearizable && !registered {
			unrendered = append(unrendered, e.Dir+" ("+e.ConcFamily+")")
		}
	}
	slices.Sort(unrendered)
	testkit.Len(t, unrendered, 0, "selects a concurrency model and no leg drives it — "+
		"render one, or register the family: "+strings.Join(unrendered, ", "))
}

// TestUnrenderedLegRegisterOnlyShrinks holds the register to its contract:
// a family the corpus stops selecting is a zombie row, and a family that
// starts rendering must lose its row in the same change.
func TestUnrenderedLegRegisterOnlyShrinks(t *testing.T) {
	t.Parallel()

	census, err := censusOnce()
	testkit.NoError(t, err, "the emission census runs")

	selected := map[string]bool{}
	for _, e := range census.Emitted {
		if e.ConcFamily != "" {
			selected[e.ConcFamily] = true
		}
	}
	var zombies []string
	for family := range gate.UnrenderedLegs {
		if !selected[family] {
			zombies = append(zombies, family)
		}
	}
	slices.Sort(zombies)
	testkit.Len(t, zombies, 0, "registered and no corpus fixture selects it — "+
		"delete the row: "+strings.Join(zombies, ", "))
}
