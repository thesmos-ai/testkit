// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package timeaware

import (
	"fmt"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/engine/model/law"
)

// MovesWithTheClock verifies that a clock-dependent answer actually
// depends on the run's clock: read it, advance, read again, and the two
// must differ.
//
// The claim `timeaware` can state on its own. Every other clock law here
// needs a quantity — how long a lifetime lasts, when a task is due — and
// the quantities belong to the classifications layered on top. What is
// left over is the dependency itself, and it is the half worth checking
// first: a subject that reads wall time instead of the clock it was
// handed passes every quantity claim its tests never advance far enough
// to break, and fails this one immediately.
//
// Only that the answer moved, not which way or by how much. Direction is
// a fact about the quantity — an age grows, a remaining lifetime
// shrinks — and this law does not know which it has.
type MovesWithTheClock[T any, K any] struct {
	// Read is the clock-dependent answer, keyed. Errors abort the law as
	// vacuous: a key nothing wrote has no reading to move.
	Read func(rt *rapid.T, sut T, k K) (int64, error)

	// Keys draws the key to read, from the run's own pool so the law asks
	// about state the sequences produced.
	Keys *rapid.Generator[K]

	// Advance moves the run's clock.
	Advance func(time.Duration)
}

// DefaultAdvance is how far [MovesWithTheClock] moves the clock.
//
// An hour rather than a moment, and fixed rather than configurable. An
// answer reported in whole seconds does not move for a millisecond, so a
// law advancing by one would read a coarse unit as a subject ignoring
// the clock — and `timeaware` names no quantity for a binding to pass
// in, which is the whole reason this is the claim it can state alone.
const DefaultAdvance = time.Hour

// ID returns the stable identifier for this law.
func (MovesWithTheClock[T, K]) ID() string { return lawid.TimeawareMoves }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (MovesWithTheClock[T, K]) REQID() string { return "" }

// Check reads, advances, and reads again.
//
// The same key both times, drawn once: two different keys would differ
// for reasons that have nothing to do with the clock, and the law would
// pass against a subject that ignored it.
func (l MovesWithTheClock[T, K]) Check(rt *rapid.T, sut, _ T) error {
	k := l.Keys.Draw(rt, "timeaware_key")

	before, err := l.Read(rt, sut, k)
	if err != nil {
		return law.Vacuous // nothing wrote this key, so there is no reading to move
	}

	l.Advance(DefaultAdvance)

	after, err := l.Read(rt, sut, k)
	if err != nil {
		return law.Vacuous // the advance took the key out of scope, which other laws own
	}
	if before == after {
		return fmt.Errorf(
			"MovesWithTheClock: the answer for this key was %d before advancing %v and %d "+
				"after; a reading that depends on the clock has to move when the clock does, "+
				"and one that does not is reading a clock the run cannot control",
			before, DefaultAdvance, after,
		)
	}
	return nil
}
