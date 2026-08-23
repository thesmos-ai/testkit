// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package crash holds the crash-recovery schedule: a drawn interleaving
// of writes, reads and crash points, with every read judged against what
// the writes acknowledged.
//
// # Why acknowledgement rather than attempt
//
// The oracle records a write only when it returned no error. That makes
// the claim two-sided and it is the whole reason the schedule can run
// against a medium that fails: a write the implementation refused is a
// write it promised nothing about, so losing it is correct — and having
// it turn up after a rebuild is as wrong as losing one it accepted. A
// schedule that recorded attempts could only state the first half, and
// would red every honest implementation that ever returned an error.
//
// # Why the crash is not a Close
//
// [Schedule.Rebuild] is handed the prior instance and nothing else. A
// crash is the goodbye that never happens: no flush, no teardown, no
// chance to write out what was buffered. Reaching for Close first would
// test the shutdown path, which is a different claim and the one an
// implementation is far more likely to have got right.
//
// The medium belongs to the instance rather than to the constructor,
// which is why Rebuild takes it — see [suite.Subject.Recover], whose
// seam this drives.
//
// [suite.Subject.Recover]: https://go.thesmos.sh/testkit/engine/suite#Subject.Recover
package crash

import (
	"context"
	"errors"

	"go.thesmos.sh/testkit/engine/model"
)

// DefaultSteps is how long a drawn schedule runs when one is not set.
//
// Long enough that several crashes land inside one run — the interesting
// state is what a rebuild inherits from a rebuild — and short enough that
// a shrunk counterexample is still readable.
const DefaultSteps = 40

// Schedule is one interface's crash-recovery verbs.
//
// Generic over the record and its identity rather than over an opaque
// operation, because the oracle has to compare a read against what a
// write installed, and a comparison needs both halves typed.
type Schedule[S any, K comparable, V any] struct {
	// Values draws the record a write installs.
	Values *model.Generator[V]

	// KeyOf is the identity a written record lands on, so the oracle
	// knows which read the write answers.
	KeyOf func(V) K

	// Keys draws the key a read asks for.
	//
	// It may overlap the keys the writes land on entirely, and usually
	// does — a generated leg hands both halves the same pool, because the
	// pool is the interface's key domain and inventing a key outside it
	// would mean inventing a literal. The absent arm still bites: every
	// schedule starts with nothing acknowledged, so each key is one a
	// rebuild must answer absent for until the first write reaches it.
	//
	// A pool wider than the writes' makes that arm bite for the whole
	// run rather than only its start, which is worth doing where a key
	// outside the domain can be named honestly.
	Keys *model.Generator[K]

	// Write installs a record. A nil error is an acknowledgement, and an
	// acknowledgement is what the oracle holds the world to across every
	// later crash.
	Write func(ctx context.Context, world S, v V) error

	// Read asks the world for a key.
	Read func(ctx context.Context, world S, k K) (V, error)

	// Equal compares a read record against the acknowledged one. Supplied
	// rather than assumed comparable: a record carrying a stamp the
	// implementation assigns is equal on the members the contract is
	// about, not on every field.
	Equal func(a, b V) bool

	// Absent reports whether a read error means the key is not there.
	// When it is nil any error counts as absent, which is the lenient
	// reading for a contract that stamps no miss sentinel.
	Absent func(error) bool

	// Rebuild is the crash seam: the world dies without warning and is
	// built again over whatever the prior instance left on its medium.
	Rebuild func(prior S) S

	// Steps bounds the drawn schedule. Zero takes [DefaultSteps].
	Steps int
}

// oracle is what the writes acknowledged, by key.
type oracle[K comparable, V any] map[K]V

// The verbs a schedule draws, named once so the weighting below and the
// switch that reads it cannot come to disagree about a spelling.
const (
	verbWrite = "write"
	verbRead  = "read"
	verbCrash = "crash"
)

// verbs is the draw the schedule steps through.
//
// Two writes to every read and every crash keeps the medium filling
// faster than the schedule empties it, so a crash late in a run has
// something to lose. A uniform draw over the three spends most of a
// short schedule reading an empty store.
//
//nolint:gochecknoglobals // a fixed weighting, read-only after init.
var verbs = model.SampledFrom([]string{
	verbWrite, verbWrite, verbRead, verbRead, verbCrash,
})

// Run drives one drawn schedule against world.
//
// wrap dresses each incarnation before the schedule touches it, and runs
// again after every crash because a fault schedule belongs to the
// incarnation rather than to the run — a medium that fails is failing
// for this boot, and a rebuild is a new boot. Pass nil for the plain
// schedule, where the world is the instance itself.
func Run[S any, K comparable, V any](
	rt *model.T, world S, sch Schedule[S, K, V], wrap func(*model.T, S) S,
) {
	if wrap == nil {
		wrap = func(_ *model.T, s S) S { return s }
	}
	steps := sch.Steps
	if steps == 0 {
		steps = DefaultSteps
	}

	sut := world
	dressed := wrap(rt, sut)
	acked := oracle[K, V]{}

	for range model.IntRange(1, steps).Draw(rt, "steps") {
		switch verbs.Draw(rt, "op") {
		case verbWrite:
			v := sch.Values.Draw(rt, "value")
			if err := sch.Write(rt.Context(), dressed, v); err == nil {
				acked[sch.KeyOf(v)] = v
			}
		case verbRead:
			k := sch.Keys.Draw(rt, "key")
			got, err := sch.Read(rt.Context(), dressed, k)
			verify(rt, sch, acked, k, got, err)
		case verbCrash:
			sut = sch.Rebuild(sut)
			dressed = wrap(rt, sut)
		}
	}
}

// verify judges one read against what the writes acknowledged.
//
// Three ways to be wrong, each naming which promise was broken. "The
// crash schedule failed" on a forty-step counterexample sends a reader
// back to the shrunk trace to work out what the claim even was.
//
// A free function rather than a method on the oracle: the schedule
// carries the comparison and the miss reading, and Go has no method type
// parameter to admit its subject type.
func verify[S any, K comparable, V any](
	rt *model.T, sch Schedule[S, K, V], acked oracle[K, V], k K, got V, err error,
) {
	want, promised := acked[k]
	switch {
	case promised && err != nil:
		rt.Fatalf("Read(%v) = %v, and that write was acknowledged: an acknowledged "+
			"write is a debt the medium owes across a rebuild", k, err)
	case promised && !sch.Equal(got, want):
		rt.Fatalf("Read(%v) = %+v, want the acknowledged %+v: the world holds a "+
			"different record from the one the write installed", k, got, want)
	case !promised && !sch.absent(err):
		rt.Fatalf("Read(%v) = (%+v, %v), want absent: nothing acknowledged this key, "+
			"so a world answering for it invented state", k, got, err)
	}
}

// absent reads a miss the schedule's way, or leniently where it stamps
// no sentinel.
func (s Schedule[S, K, V]) absent(err error) bool {
	if err == nil {
		return false
	}
	if s.Absent == nil {
		return true
	}
	return s.Absent(err)
}

// Is builds a [Schedule.Absent] from a sentinel, for the ordinary case
// where a contract stamps one.
func Is(sentinel error) func(error) bool {
	return func(err error) bool { return errors.Is(err, sentinel) }
}
