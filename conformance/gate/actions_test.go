// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"reflect"
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
	"go.thesmos.sh/testkit/engine/model/ref"
	"go.thesmos.sh/testkit/generator/core/tiers"
)

// TestEveryActionRowNamesAShippedConstructor closes the action table's engine
// side.
//
// The tiers side already holds every detector to a row; this holds every row's
// answer to a function `engine/model/action` exports. Between the two, the
// table cannot name a constructor that does not ship and cannot miss a shape
// that does — the drift the last hand-maintained mapping was retired for.
func TestEveryActionRowNamesAShippedConstructor(t *testing.T) {
	t.Parallel()

	for _, d := range detectors.All() {
		ctor, ok := tiers.ActionFor(d.Name)
		testkit.True(t, ok, d.Name+" has a row")

		fn, shipped := gate.ActionCtors[ctor]
		testkit.True(t, shipped, d.Name+"'s constructor "+ctor+" is in the census")
		if !shipped {
			continue
		}
		testkit.Equal(t, reflect.TypeOf(fn).Kind(), reflect.Func,
			ctor+" is a function")
	}
}

// TestCensusCarriesNoRetiredConstructor is the other direction: an entry here
// that no detector's row reaches is a constructor the census claims is in use
// and is not — the stale-excuse problem, one table over.
func TestCensusCarriesNoRetiredConstructor(t *testing.T) {
	t.Parallel()

	reached := map[string]bool{}
	for _, d := range detectors.All() {
		if ctor, ok := tiers.ActionFor(d.Name); ok {
			reached[ctor] = true
		}
	}
	// The contract-role rows reach constructors no detector names: the role
	// table re-points them after the shape chose, and they are in use for
	// exactly as long as a row says so.
	for key, ctor := range tiers.ContractActionRows() {
		reached[ctor] = shippedCtor(t, key, ctor)
	}
	// The recording rows reach a third set: a law reading the run's write
	// log swaps the shape's constructor for the one that files what it
	// wrote, and those are in use for exactly as long as a row says so.
	for shape, ctor := range tiers.RecordingActionRows() {
		reached[ctor] = shippedCtor(t, shape, ctor)
	}
	for name := range gate.ActionCtors {
		testkit.True(t, reached[name], name+" is reached by some detector's row")
	}
}

// shippedCtor holds one table row to a real constructor, and reports that
// it reached one.
//
// Shared by the two row loops because the demand is one: a table naming a
// constructor the engine does not ship renders a call to nothing, and the
// generated file is where that would be discovered.
func shippedCtor(t *testing.T, key, ctor string) bool {
	t.Helper()
	fn, shipped := gate.ActionCtors[ctor]
	testkit.True(t, shipped, key+"'s constructor "+ctor+" is in the census")
	if !shipped {
		return false
	}
	testkit.Equal(t, reflect.TypeOf(fn).Kind(), reflect.Func, ctor+" is a function")
	return true
}

// TestEveryMapStoreOpIsAMethod holds the oracle delegation rows to the shipped
// oracle.
//
// The generated adapter forwards a shape's parameters in order and changes
// only the name, which is sound exactly while the named method exists with the
// shape's own signature. Existence is checked here; the signature agreement is
// checked where it cannot lie — the corpus, which compiles the adapter.
func TestEveryMapStoreOpIsAMethod(t *testing.T) {
	t.Parallel()

	// Both map forms carry the same surface: the refinement swaps the store,
	// not the delegation table, so every op a shape reaches on the plain map
	// must exist on the pinning one.
	store := reflect.TypeFor[ref.MapStore[string, string]]()
	sticky := reflect.TypeFor[ref.StickyStore[string, string]]()
	for _, s := range tiers.MapStoreShapes() {
		op, _ := tiers.MapStoreOp(s)
		testkit.True(t, gate.HasMethod(store, op),
			s+" delegates to MapStore."+op+", which exists")
		testkit.True(t, gate.HasMethod(sticky, op),
			s+" delegates to "+op+" on the pinning form too")
	}
	testkit.True(t, tiers.MapStorePins("sticky"),
		"the pinning refinement follows the claim")
	testkit.False(t, tiers.MapStorePins("validates"), "and only the claim")

	keyed := reflect.TypeFor[ref.KeyedStore[string, string]]()
	for _, s := range tiers.KeyedStoreShapes() {
		op, _ := tiers.KeyedStoreOp(s)
		testkit.True(t, gate.HasMethod(keyed, op),
			s+" delegates to KeyedStore."+op+", which exists")
	}
	op, assigned := tiers.KeyedStoreMixinOp("deleteremoves")
	testkit.True(t, assigned && gate.HasMethod(keyed, op),
		"the delete assignment names a method the keyed oracle declares")

	// Both collection forms carry the same surface, so one op table serves
	// the log and the set the dedupe claim refines it into.
	log := reflect.TypeFor[ref.Collection[string]]()
	set := reflect.TypeFor[ref.SetCollection[string]]()
	for _, s := range []string{"writer", tiers.ShapeCollector} {
		op, ok := tiers.CollectionOp(s)
		testkit.True(t, ok, s+" is a shape the collection models")
		testkit.True(t, gate.HasMethod(log, op) && gate.HasMethod(set, op),
			s+" delegates to "+op+", which both forms declare")
	}
	testkit.True(t, tiers.CollectionDedupes("noduplicates"),
		"the dedupe refinement follows the claim")
	testkit.False(t, tiers.CollectionDedupes("permutation"),
		"and only the claim")
}
