// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
	"go.thesmos.sh/testkit/generator/core/tiers"
)

// assertionState is the one owed-versus-bound measurement this file's tests
// share: an Annotate run for the stamps, an Emission run for the bindings —
// each a full pipeline over the corpus, too expensive to repeat per test.
type assertionState struct {
	owed  map[string]bool
	bound map[string]bool
	twins int
	err   error
}

// censusOnce is the single corpus run every measurement in this package reads.
//
// One run rather than one per question. `go/types` memoizes an Alias without
// synchronization and the package loader is concurrent, so each full corpus
// load is another chance for the detector to pair two accesses to it — the
// GOMAXPROCS pin in TestMain narrows that window and does not close it. This
// package had grown to six independent loads and a -race run found the race
// they add up to.
//
//nolint:gochecknoglobals // memoized measurement, test-only.
var censusOnce = sync.OnceValues(func() (gate.Census, error) {
	return gate.Measure(context.Background(), corpusRoot, "./corpus/...")
})

//nolint:gochecknoglobals // memoized measurement, test-only.
var assertionOnce = sync.OnceValue(func() assertionState {
	s := assertionState{owed: map[string]bool{}, bound: map[string]bool{}}
	stamped, err := gate.Annotate(context.Background(), corpusRoot, "./corpus/...")
	if err != nil {
		s.err = err
		return s
	}
	census, err := censusOnce()
	emitted := census.Emitted
	if err != nil {
		s.err = err
		return s
	}
	for _, names := range stamped {
		for _, c := range names {
			for _, law := range tiers.LawsFor(c) {
				// An unsound conduct cannot bind by design; the conduct
				// census owns that quarantine, not this register.
				if gate.LawConduct[law].Sound() {
					s.owed[law] = true
				}
			}
		}
	}
	for _, e := range emitted {
		for _, law := range e.Laws {
			s.bound[law] = true
		}
		if e.Twin {
			s.twins++
		}
	}
	return s
})

// TestEveryOwedLawIsBoundOrRegistered is the assertion gate the audit
// commissioned: a classification stamped in the corpus selects laws, and
// each selected, sound law must be bound in at least one fixture — or carried
// in [gate.UnboundLaws] with the chokepoint that holds it. The bounded break
// experiment proved what the gap costs: a fixture's whole claim deleted, the
// corpus green. A red line here names the law that would go the same way.
func TestEveryOwedLawIsBoundOrRegistered(t *testing.T) {
	t.Parallel()

	s := assertionOnce()
	if s.err != nil {
		t.Fatalf("measure the corpus: %v", s.err)
	}
	testkit.True(t, len(s.owed) > 0, "the corpus stamps select laws at all")
	// Every one, not the first: a census that stops at the first gap makes
	// closing the rest a run apiece, and the whole point of measuring the
	// set is to see its size.
	var unbound []string
	for law := range s.owed {
		if _, registered := gate.UnboundLaws[law]; !registered && !s.bound[law] {
			unbound = append(unbound, law)
		}
	}
	slices.Sort(unbound)
	testkit.Len(t, unbound, 0, "selected by the corpus and bound nowhere — bind each, "+
		"or register the debt with its chokepoint: "+strings.Join(unbound, ", "))
}

// TestUnboundRegisterOnlyShrinks holds the register to its contract: an entry
// that starts binding must be deleted, and an entry nothing selects any more
// is a zombie. Either way the register moves in one direction.
func TestUnboundRegisterOnlyShrinks(t *testing.T) {
	t.Parallel()

	s := assertionOnce()
	if s.err != nil {
		t.Fatalf("measure the corpus: %v", s.err)
	}
	for law, reason := range gate.UnboundLaws {
		testkit.False(t, s.bound[law],
			law+" now binds — delete its register entry; the debt is paid")
		testkit.True(t, s.owed[law],
			law+" is owed by nothing the corpus stamps — a zombie entry records no debt")
		testkit.True(t, len(reason) > 30,
			law+"'s reason says what it is waiting on")
	}
}

// TestEmissionSeesTheTwinFloor pins the measurement's other axis: the
// reference kind is read off the queued bindings, so the twin count the
// audit hand-derived stays derivable — and a derived fixture regressing to
// the twin is visible here before it is visible nowhere.
func TestEmissionSeesTheTwinFloor(t *testing.T) {
	t.Parallel()

	emitted, err := gate.Emission(t.Context(), corpusRoot,
		"./corpus/iface/mixin/validates", "./corpus/iface/mixin/bounded")
	if err != nil {
		t.Fatalf("measure two fixtures: %v", err)
	}
	kinds := map[string]bool{}
	for _, e := range emitted {
		kinds[e.Fixture] = e.Twin
	}
	testkit.False(t, kinds["go.thesmos.sh/testkit/conformance/corpus/iface/mixin/validates.Mixed"],
		"validates derives the map oracle")
	testkit.True(t, kinds["go.thesmos.sh/testkit/conformance/corpus/iface/mixin/bounded.Mixed"],
		"bounded rides the twin floor — the audit's break experiment, kept measurable")
}

// twinCeiling is how many corpus references ride the twin floor.
//
// The number only sinks. An oracle upgrade lowers it; a derived fixture
// regressing to the twin pushes past it and reddens this build by name.
// Lower the constant in the same commit that lowers the count.
//
// Raise it only for a fixture whose floor is ARGUED, and put the argument
// in the table. Every row there is a claim about what no store oracle can
// model, not a gap waiting on somebody's time.
//
//	fixture                 why no derived oracle can answer
//	----------------------- ------------------------------------------
//	scheduled               a schedule beside a firing count derives no
//	                        map; twins on one clock fire together
//	session (5)             the subject assigns the version member on
//	                        write, which no value-storing oracle stamps
//	answeringwriter         a lone writer derives nothing to compare
//	                        through
//	publisher modes (4)     the delivery laws mirror the whole cycle onto
//	                        the reference and then read the subject's
//	                        drain alone, so what stands behind ref
//	                        changes nothing any check reads
//	idempotentclose         a teardown beside an open-count aggregate
//	                        derives no store at all
//	atomic                  the claim is about refused writes, and a
//	                        derived map refuses nothing
//	causal-chain            the claim is an admission policy, and a
//	                        derived log admits everything
//	bounded, batched-mixins the claim clamps what a COLLECTION reader
//	                        answers, and a derived one clamps nothing
//	indexed                 its reader addresses a position, and every
//	                        store oracle addresses a key
//	timeaware               its reader answers how long ago a key was
//	                        seen, a fact about the clock rather than
//	                        about anything stored
//
// The bounded row is narrower than it reads: a bound over a KEYED read
// that may say no derives an unbounded oracle and compares hits one way.
// See [tiers.OracleDefeat], and boundedcache, which is the fixture that
// does.
//
// The count rose to 74 when the differential started being emitted for a
// twin reference. That is a measurement change rather than a regression:
// the row had never been emitted for a twin at all, so a fixture binding
// no law had an empty model tier — and an empty tier is not counted here.
// Eight detector fixtures were riding this floor invisibly, and with them
// the action constructors for a lookup, a multi-reader, a mutator and a
// bool-answering read were built and never driven anywhere in the corpus.
//
// What the row claims there is small and stated: two instances of one
// subject must agree, which catches nondeterminism and hidden shared
// state. It ships Argued, because no planted defect can redden it — a
// proof run builds both sides from the defect and they agree however
// broken it is.
//
// The publisher row left this table when the family gained ref.FanOut, an
// oracle written at the shapes a Go publisher declares rather than at a
// queue a consumer drains. Five fixtures took it, and five more stopped
// being counted at all: their only model row was a count compared against
// their own factory, and a row that can fail on nothing but
// nondeterminism is declined rather than shipped. Those five now report
// an empty tier, which is what they were checking.
const twinCeiling = 74

// TestTwinFloorOnlySinks is the twin-count ratchet the audit's second item
// commissioned: the twin is the honest floor, not the resting state, and a
// regression from a derived oracle back to it must be visible somewhere
// before it is visible nowhere.
func TestTwinFloorOnlySinks(t *testing.T) {
	t.Parallel()

	s := assertionOnce()
	if s.err != nil {
		t.Fatalf("measure the corpus: %v", s.err)
	}
	testkit.True(t, s.twins <= twinCeiling,
		"the twin floor only sinks — a derived fixture regressed, or a new fixture "+
			"needs its oracle argued for before it rides the floor")
	// Equal rather than True, so the message carries the count: "lower it to
	// the new number" is advice nobody can act on without being told the
	// number, and finding it meant instrumenting this test by hand.
	testkit.Equal(t, s.twins, twinCeiling,
		"the floor sank — lower twinCeiling to the count on the left and bank the progress")
}

// TestEmissionSurfacesARunFailure pins the error arm: a pattern matching
// nothing is a run that failed, not an empty measurement quietly read as
// "nothing owed".
func TestEmissionSurfacesARunFailure(t *testing.T) {
	t.Parallel()

	_, err := gate.Emission(t.Context(), corpusRoot, "./corpus/definitely-not-here/...")
	testkit.True(t, err != nil, "a failed run reports, never measures empty")
}
