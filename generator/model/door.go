// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/sdk"
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
const (
	KindClockDoor      sdk.Kind = "model.door.clock"
	KindClockLowering  sdk.Kind = "model.lowering.clock"
	KindInduceDoor     sdk.Kind = "model.door.induce"
	KindInduceLowering sdk.Kind = "model.lowering.induce"
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
	// that does not involve one.
	Clock string

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

// InduceDoor is the map a check about a failure state needs, so it can
// provoke the state before asking what happens in it. InduceLowering
// carries it onto the runtime.
type InduceDoor struct{ door }

// Kind returns the template this contribution renders through.
func (*InduceDoor) Kind() sdk.Kind { return KindInduceDoor }

// InduceLowering is [InduceDoor]'s other half.
type InduceLowering struct{ door }

// Kind returns the template this contribution renders through.
func (*InduceLowering) Kind() sdk.Kind { return KindInduceLowering }

// doorsFor is every field this interface's bound laws need, each with
// the line that carries it.
//
// Read off the bindings rather than the classifications, because a field
// is owed by a law that actually BOUND. A law selected and then refused
// states nothing, so a field for it would be one a consumer must fill
// for a check that never runs.
//
// The pair is the unit: a field nothing lowers is somewhere to write a
// value that goes nowhere, which is what the harness carried for a clock
// before either existed.
func doorsFor(b *Bindings, subject string) (fields, lowerings []sdk.EmitNode) {
	d := door{Vocab: VocabPkg, Clock: ClockPkg, Subject: subject}

	for _, l := range b.Laws {
		if l.Clocked {
			fields = append(fields, &ClockDoor{door: d})
			lowerings = append(lowerings, &ClockLowering{door: d})
			break
		}
	}
	if b.Induces() {
		fields = append(fields, &InduceDoor{door: d})
		lowerings = append(lowerings, &InduceLowering{door: d})
	}
	return fields, lowerings
}
