// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/sdk"
)

// KindClockDoor is the emit kind and the template name for this tier's
// clock contribution to the harness.
const KindClockDoor sdk.Kind = "model.door.clock"

// KindClockLowering is the emit kind and template for the line that
// carries the clock constructors onto the runtime subject.
const KindClockLowering sdk.Kind = "model.lowering.clock"

// ClockDoor is what a clocked check needs a consumer to supply, rendered
// into the harness the other generator emits.
//
// A kind per capability, and the template renders the Go. The alternative
// — one Door type carrying a name, a type and some prose — models a field
// and this is not one field: the clock arrives as a pair of constructors
// mirroring New and Start, and it needs a line in Subject to lower it. A
// structure describing "a field" could express neither, which is how it
// went in as a bare `OnClock clock.Clock` that nothing reads.
//
// So the type carries only what the template cannot work out: which
// package spells the clock. Everything else — the two fields, their
// prose, the exclusivity guard, the lowering — is Go, and Go belongs in
// a template.
type ClockDoor struct {
	sdk.BaseEmit

	// Pkg is the clock package's import path. Named rather than
	// resolved in the template because `external` takes a path, and the
	// path is this tier's to know: the harness generator has no clock.
	Pkg string

	// Subject is the interface the harness is for, in type position, so
	// the lowering's closures spell their return.
	Subject sdk.Ref
}

// Kind returns the template this contribution renders through.
func (*ClockDoor) Kind() sdk.Kind { return KindClockDoor }

// ClockLowering carries the harness's clock constructors onto the
// runtime subject. Its own kind because it renders into a different
// region of the same file — the body of Subject rather than the struct.
type ClockLowering struct {
	sdk.BaseEmit

	// Pkg and Subject are [ClockDoor]'s, for the same reasons.
	Pkg     string
	Subject sdk.Ref

	// Vocab is the runtime suite package, whose Subject type the guard
	// returns and whose ExclusivePair refuses a harness that set both
	// constructors.
	Vocab string
}

// Kind returns the template this contribution renders through.
func (*ClockLowering) Kind() sdk.Kind { return KindClockLowering }

// clockDoorFor is the contribution for an interface whose bound laws
// move time, nil where none do.
//
// Read off the bindings rather than the classifications, because the
// field is owed by a law that actually BOUND. A law selected and then
// refused states nothing, so a field for it would be one a consumer must
// fill for a check that never runs.
func clockDoorFor(b *Bindings, subject sdk.Ref) (*ClockDoor, *ClockLowering) {
	for _, l := range b.Laws {
		if l.Clocked {
			return &ClockDoor{Pkg: ClockPkg, Subject: subject},
				&ClockLowering{Pkg: ClockPkg, Subject: subject, Vocab: VocabPkg}
		}
	}
	return nil, nil
}
