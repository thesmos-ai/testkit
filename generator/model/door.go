// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"
)

// KindDoor is the emit kind and the template name for a harness field
// this tier asks the harness generator to carry.
const KindDoor sdk.Kind = "model.door"

// Door is one capability field contributed into the harness.
//
// The harness is the other generator's output and it renders whatever
// is in its doors region without reading it. That is the right way
// round: the field exists because a check in THIS tier cannot state its
// claim without it, so the sentence telling a consumer why they must
// fill it is this tier's sentence to write. A harness generator
// composing that prose would be explaining a check it did not emit.
type Door struct {
	sdk.BaseEmit

	// Name is the field's identifier on the harness.
	Name string

	// Type is the field's type, rendered through the backend so the
	// harness file registers whatever import it needs. The import
	// travels with the contribution: a package whose interfaces carry no
	// model directive contributes no door and imports nothing.
	Type sdk.Ref

	// Why is the docblock, one line per element, without the comment
	// marker — the template spells that.
	Why []string
}

// Kind returns the template this door renders through.
func (*Door) Kind() sdk.Kind { return KindDoor }

// clockDoor is the field a clocked check needs.
//
// Every claim about time in this tier moves the clock rather than
// waiting on it, which is the only way the claim is stateable at all: an
// implementation reading the real clock would have to be waited on in
// real time, so the check would either take minutes or assert nothing.
func clockDoor(clock sdk.Ref) *Door {
	return &Door{
		Name: "OnClock",
		Type: clock,
		Why: []string{
			"OnClock builds an instance reading the given clock. Checks that move",
			"time need it, and fail naming this field when it is nil.",
			"",
			"A constructor rather than a clock handed to a built instance: an",
			"implementation that reads the clock at construction cannot be told",
			"about a different one afterwards, and a check that moved time would",
			"be moving a clock the subject never looks at.",
		},
	}
}

// doorsOf is every harness field this interface's bound laws need.
//
// Read off the bindings rather than off the classifications, because a
// door is owed by a law that actually BOUND. A law selected and then
// refused states nothing, so a field for it would be one a consumer
// must fill for a check that never runs — which is the reading
// [projection.HarnessOf] already takes of this generator's own rows.
func doorsOf(b *Bindings) []*Door {
	for _, l := range b.Laws {
		if l.Clocked {
			return []*Door{clockDoor(golang.RefFor("Clock", ClockPkg))}
		}
	}
	return nil
}
