// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"slices"

	"go.thesmos.sh/eidos/sdk"

	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// The emit kinds and template names for what this tier contributes to
// the harness: a field, and the line in Subject that carries it onto the
// runtime.
//
// A kind per capability, and the template renders the Go. One Door type
// carrying a name, a type and some prose would model a field, and these
// are not one field each: the clock arrives as a pair of constructors
// mirroring New and Start, the induction as a map behind a type alias,
// and both need a line in Subject. A structure describing "a field"
// could express none of it, which is how the clock first went in as a
// bare value nothing read.
// Three capabilities, because three are what this tier's checks ask for.
// A clocked law moves time; a crash schedule needs the process built
// again over the medium the prior one left, and needs the medium itself
// able to fail. Every other leg provokes what it needs through methods
// the interface already declares — the poison law calls the failing
// method, the lifecycle law calls Close — and a field for a state the
// checks reach without it is a field a consumer fills for nothing.
const (
	KindClockDoor       sdk.Kind = "model.door.clock"
	KindClockLowering   sdk.Kind = "model.lowering.clock"
	KindInduceDoor      sdk.Kind = "model.door.induce"
	KindInduceLowering  sdk.Kind = "model.lowering.induce"
	KindRecoverDoor     sdk.Kind = "model.door.recover"
	KindRecoverLowering sdk.Kind = "model.lowering.recover"
)

// door is what every contribution to the harness needs and no template
// can work out for itself.
type door struct {
	sdk.BaseEmit

	// Vocab is the runtime suite package: the Subject a lowering returns,
	// the guards it calls, the capability types a field spells. Named
	// here because the harness generator's own templates reach it through
	// a helper this one has no access to.
	Vocab string

	// Clock is the controllable clock's package, empty for a capability
	// that does not involve one, and LegsPkg the bridge whose narrowing
	// the crash seam's lowering reaches for.
	Clock, LegsPkg string

	// Subject is the interface as the harness file spells it, from
	// [suite.Contract.SubjectType]. Taken rather than derived because the
	// qualified form compiles beside the local one, so a second
	// derivation would put two spellings of one type in one file and
	// nothing would complain.
	Subject string
}

// ClockDoor is the constructor pair a clocked check needs a consumer to
// supply. ClockLowering carries whichever is set onto the runtime.
type ClockDoor struct{ door }

// Kind returns the template this contribution renders through.
func (*ClockDoor) Kind() sdk.Kind { return KindClockDoor }

// ClockLowering is [ClockDoor]'s other half.
type ClockLowering struct{ door }

// Kind returns the template this contribution renders through.
func (*ClockLowering) Kind() sdk.Kind { return KindClockLowering }

// InduceDoor is the trigger table a check needs when it asks for a
// subject in a named failure state. InduceLowering carries it onto the
// runtime.
type InduceDoor struct{ door }

// Kind returns the template this contribution renders through.
func (*InduceDoor) Kind() sdk.Kind { return KindInduceDoor }

// InduceLowering is [InduceDoor]'s other half.
type InduceLowering struct{ door }

// Kind returns the template this contribution renders through.
func (*InduceLowering) Kind() sdk.Kind { return KindInduceLowering }

// RecoverDoor is the constructor pair a crash schedule needs a consumer
// to supply: the process built again over the medium the prior instance
// left. RecoverLowering carries whichever is set onto the runtime.
type RecoverDoor struct{ door }

// Kind returns the template this contribution renders through.
func (*RecoverDoor) Kind() sdk.Kind { return KindRecoverDoor }

// RecoverLowering is [RecoverDoor]'s other half.
type RecoverLowering struct{ door }

// Kind returns the template this contribution renders through.
func (*RecoverLowering) Kind() sdk.Kind { return KindRecoverLowering }

// doorsFor is every field this interface's rows need, each with the line
// that carries it.
//
// Read off the rows rather than off the laws, and that is the whole rule:
// a field is owed by a CHECK, and a check is what a row is. A law can
// bind and still reach no row — its claim may go unworded, or its leg may
// wait on a closure only the consumer has — and a field for it is one a
// consumer must fill for a check that never comes. The corpus had exactly
// that: a windowed harness asking for a clock, with no clocked check in
// the file.
//
// The pair is the unit: a field nothing lowers is somewhere to write a
// value that goes nowhere, which is what the harness carried for a clock
// before either existed.
func doorsFor(b *Bindings, subject string) (fields, lowerings []sdk.EmitNode) {
	d := door{Vocab: VocabPkg, Clock: ClockPkg, LegsPkg: LegsPkg, Subject: subject}
	if anyRowNeeds(b, vocab.CapClock) {
		fields = append(fields, &ClockDoor{door: d})
		lowerings = append(lowerings, &ClockLowering{door: d})
	}
	if anyRowNeeds(b, vocab.CapInduce) {
		fields = append(fields, &InduceDoor{door: d})
		lowerings = append(lowerings, &InduceLowering{door: d})
	}
	if anyRowNeeds(b, vocab.CapRecover) {
		fields = append(fields, &RecoverDoor{door: d})
		lowerings = append(lowerings, &RecoverLowering{door: d})
	}
	return fields, lowerings
}

// anyRowNeeds reports that some row this tier planned demands the given
// door of the harness.
//
// One pass per capability rather than one switch over every row: the
// contributions land in a fixed order this way, and a generated harness
// whose fields reshuffle between runs is a diff nobody can review.
func anyRowNeeds(b *Bindings, c vocab.Capability) bool {
	return slices.ContainsFunc(b.Rows, func(r projection.CheckPlan) bool {
		return slices.ContainsFunc(r.Needs, func(n projection.NeedPlan) bool {
			return n.Capability == c
		})
	})
}
