// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/core/tiers"
)

// TestEveryDetectorDrivesAnAction holds the action table total over the live
// registry.
//
// A detector without a row is a method silently absent from every generated
// sequence — the run still passes, over sequences that never call it. eidos
// adding a detector must fail this test by name, in the same build that makes
// the classification stampable.
func TestEveryDetectorDrivesAnAction(t *testing.T) {
	t.Parallel()

	for _, d := range detectors.All() {
		ctor, ok := tiers.ActionFor(d.Name)
		testkit.True(t, ok, d.Name+" names an action constructor")
		testkit.True(t, ctor != "", d.Name+"'s constructor is not empty")
	}
}

// TestActionForDeclinesTheUnknown pins the miss arm: a name outside the
// vocabulary answers false rather than an empty constructor a template would
// render as a call to nothing.
func TestActionForDeclinesTheUnknown(t *testing.T) {
	t.Parallel()

	ctor, ok := tiers.ActionFor("not-a-shape")
	testkit.False(t, ok, "an unregistered shape names no constructor")
	testkit.Equal(t, ctor, "", "and returns nothing to render")
}

// TestMapStoreOpsAreDetectorShapes holds the oracle's delegation rows to the
// same vocabulary as the action rows.
//
// A row keyed on a name no detector stamps is delegation nothing can reach;
// the adapter renders every stamped method either through a row or inert, so
// an unreachable row is a modelled shape that silently became inert.
func TestMapStoreOpsAreDetectorShapes(t *testing.T) {
	t.Parallel()

	live := map[string]bool{tiers.ShapeCollector: true}
	for _, d := range detectors.All() {
		live[d.Name] = true
		// The pseudo-shape must never collide with a real detector: it is
		// the generator's own refinement, and a detector landing upstream
		// under the same spelling would make one name mean two things.
		testkit.NotEqual(t, d.Name, tiers.ShapeCollector,
			"the collector pseudo-shape stays outside the detector vocabulary")
	}
	for _, s := range tiers.MapStoreShapes() {
		op, ok := tiers.MapStoreOp(s)
		testkit.True(t, ok, s+" is a shape the oracle models")
		testkit.True(t, op != "", s+"'s delegation names a method")
		testkit.True(t, live[s], s+" is a shape the annotator stamps or the generator derives")
	}
}

// TestKeyedStoreDelegation pins the keyed oracle's rows: the four shapes it
// models, the census over them, and the one mixin-assigned method whose
// semantics no signature carries.
func TestKeyedStoreDelegation(t *testing.T) {
	t.Parallel()

	for shape, op := range map[string]string{
		"reader": "Get", "readerwithbool": "Get",
		"compositewriter": "Put", "aggregator": "Count",
	} {
		got, ok := tiers.KeyedStoreOp(shape)
		testkit.True(t, ok, shape+" has a keyed-oracle row")
		testkit.Equal(t, got, op, shape+" delegates to "+op)
	}
	_, ok := tiers.KeyedStoreOp("writer")
	testkit.False(t, ok, "a plain writer has no row — a keyed store cannot place a keyless value")

	// Two reads share the Get row and they are not the same delegation: the
	// bool-answering one folds the oracle's miss sentinel into its flag,
	// which the adapter does rather than this table. See AdapterMethod.Folds.
	testkit.Equal(t, len(tiers.KeyedStoreShapes()), 4, "the census names exactly the modeled shapes")

	op, ok := tiers.KeyedStoreMixinOp("deleteremoves")
	testkit.True(t, ok, "deleteremoves assigns its carrier a method")
	testkit.Equal(t, op, "Delete", "the delete is the one row a shape alone cannot earn")
	_, ok = tiers.KeyedStoreMixinOp("idempotent")
	testkit.False(t, ok, "other mixins assign nothing")
}

// TestCollectionDelegation pins the append-and-drain oracle's rows and the
// mixin refinements that pick its variants.
func TestCollectionDelegation(t *testing.T) {
	t.Parallel()

	got, ok := tiers.CollectionOp("writer")
	testkit.True(t, ok, "a value writer appends")
	testkit.Equal(t, got, "Add", "through Add")
	got, ok = tiers.CollectionOp(tiers.ShapeCollector)
	testkit.True(t, ok, "the collector drains")
	testkit.Equal(t, got, "Items", "through Items")
	_, ok = tiers.CollectionOp("reader")
	testkit.False(t, ok, "a keyed reader is the keyed oracle's territory")

	testkit.True(t, tiers.CollectionDedupes("noduplicates"),
		"noduplicates turns the collection into its deduplicating form")
	testkit.False(t, tiers.CollectionDedupes("idempotent"),
		"other mixins leave the log plain")
}

// TestOracleRefinements pins the classification-driven oracle switches: the
// history vocabularies, the claims no immediate store models, and the
// resolution pin.
func TestOracleRefinements(t *testing.T) {
	t.Parallel()

	testkit.True(t, tiers.DrainsHistory("snapshotisolation"), "snapshotisolation records events")
	testkit.True(t, tiers.DrainsHistory("chain"), "chain replays an append-only log")
	testkit.False(t, tiers.DrainsHistory("noduplicates"), "a deduped collection is holdings, not history")

	for _, mixin := range []string{"eventually", "crdtmerge"} {
		defeat, defeated := tiers.DefeatsOracles(mixin)
		testkit.True(t, defeated, mixin+" puts the subject beyond any immediate store model")
		testkit.True(t, defeat.Why != "", mixin+"'s header prints the reason")
		testkit.False(t, defeat.LiftedByEvictingRead,
			mixin+" is defeated by what the subject DOES, which no read shape undoes")
	}
	_, defeated := tiers.DefeatsOracles("idempotent")
	testkit.False(t, defeated, "idempotent claims nothing about immediacy")

	// The one conditional defeat: a bound over a collection is final, and a
	// bound over a keyed read that may say no is not. See [tiers.OracleDefeat].
	bound, defeated := tiers.DefeatsOracles("bounded")
	testkit.True(t, defeated, "a bound defeats the store model by default")
	testkit.True(t, bound.LiftedByEvictingRead,
		"and a reader that may legally say no lifts it")

	testkit.True(t, tiers.MapStorePins("sticky"), "sticky pins the first resolution")
	testkit.False(t, tiers.MapStorePins("idempotent"), "other mixins leave the map latest-write-wins")
}
