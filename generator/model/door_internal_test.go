// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// A door is owed by a check, and a check is what a row is.
//
// Read off the rows rather than off the laws, because a law can bind and
// still reach no row: its claim may go unworded, or its leg may wait on
// something only the consumer has. The corpus had exactly that — a
// windowed harness asking for a clock with no clocked check in the file,
// which is a field somebody fills for nothing.
func TestADoorFollowsTheRowThatNeedsIt(t *testing.T) {
	t.Parallel()

	clocked := projection.CheckPlan{
		Needs: []projection.NeedPlan{{Capability: vocab.CapClock}},
	}

	t.Run("a row needing a clock opens one", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Rows: []projection.CheckPlan{clocked}}
		fields, lowerings := doorsFor(b, "Mixed")
		testkit.Len(t, fields, 1, "the field a clocked check needs")
		testkit.Len(t, lowerings, 1,
			"and the line that carries it — a field nothing lowers is somewhere "+
				"to write a value that goes nowhere")
	})

	t.Run("a bound law that reached no row opens nothing", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Laws: []*LawBinding{{ID: "AUTO-TTL-EXPIRY", Clocked: true}}}
		fields, lowerings := doorsFor(b, "Mixed")
		testkit.Len(t, fields, 0,
			"a field for a check that never comes is one a consumer fills for nothing")
		testkit.Len(t, lowerings, 0, "and its other half with it")
	})

	t.Run("two clocked rows share one door", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Rows: []projection.CheckPlan{clocked, clocked}}
		fields, _ := doorsFor(b, "Mixed")
		testkit.Len(t, fields, 1, "one clock, however many checks move it")
	})
}

// Each capability opens its own door, and the two arrive in a fixed
// order.
//
// Two now, and the second is what made the order worth pinning: a
// generated harness whose fields reshuffle between runs is a diff nobody
// can review, and the clock came first only because it was first
// written. A row needing both gets both, because a check that moves time
// and kills the process needs the field for each.
func TestEachCapabilityOpensItsOwnDoor(t *testing.T) {
	t.Parallel()

	clocked := projection.CheckPlan{
		Needs: []projection.NeedPlan{{Capability: vocab.CapClock}},
	}
	crashing := projection.CheckPlan{
		Needs: []projection.NeedPlan{{Capability: vocab.CapRecover}},
	}

	t.Run("a row needing recovery opens the crash seam", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Rows: []projection.CheckPlan{crashing}}
		fields, lowerings := doorsFor(b, "Mixed")
		testkit.Len(t, fields, 1, "the field a crash schedule needs")
		testkit.Equal(t, fields[0].Kind(), KindRecoverDoor,
			"and it is the crash seam, not the clock")
		testkit.Len(t, lowerings, 1, "with the line that carries it onto the runtime")
	})

	t.Run("a clocked row opens no crash seam", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Rows: []projection.CheckPlan{clocked}}
		fields, _ := doorsFor(b, "Mixed")
		testkit.Len(t, fields, 1, "one door, for the one capability asked for")
		testkit.Equal(t, fields[0].Kind(), KindClockDoor,
			"a door opens because a check asks, never because another door did")
	})

	t.Run("both are opened in declaration order", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Rows: []projection.CheckPlan{crashing, clocked}}
		fields, lowerings := doorsFor(b, "Mixed")
		testkit.Len(t, fields, 2, "a field for each capability the rows demand")
		testkit.Equal(t, fields[0].Kind(), KindClockDoor,
			"the clock first whichever row asked first, so the harness does not reshuffle")
		testkit.Equal(t, fields[1].Kind(), KindRecoverDoor, "then the crash seam")
		testkit.Len(t, lowerings, 2, "and both halves of both")
	})
}
