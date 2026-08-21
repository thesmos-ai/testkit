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
