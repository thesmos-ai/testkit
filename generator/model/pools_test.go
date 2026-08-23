// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/plugins/annotator/shape"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/model"
)

// TestValuePoolWidth walks the widening decision's arms: the license the
// claims grant, the pin that keeps a wide draw colliding, and the two ways a
// pool stays narrow without a restricting claim.
func TestValuePoolWidth(t *testing.T) {
	t.Parallel()

	t.Run("a scalar-fielded value goes wide, keyed from the pool", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc",
			field{"ID", "string"}, field{"N", "int"}))
		testkit.True(t, b.Values.Wide, "nothing in the claims restricts the domain")
		testkit.Equal(t, b.Values.Pin, "ID", "and every draw lands on a pooled key")
		testkit.Equal(t, b.Values.WhyNarrow, "", "so there is nothing to explain")
	})

	t.Run("an unexported field is skipped the way Make skips it", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc",
			field{"ID", "string"}, field{"body", "string"}))
		testkit.True(t, b.Values.Wide, "Make leaves it zero, which draws fine")
	})

	t.Run("a field out of reach keeps the pair, pinned", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc",
			field{"ID", "string"}, field{"When", "time.Time"}))
		testkit.False(t, b.Values.Wide, "a wide draw would arm a run-time panic")
		testkit.Assert(t, b.Values.WhyNarrow).Contains("time.Time", "naming the reach")
		testkit.Equal(t, b.Values.Pin, "ID",
			"recombining proven bodies with pooled keys is still licensed")
	})

	t.Run("a keyed put widens with no pin", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, keyedStore(t, "example.com/kv.ErrGone"))
		testkit.True(t, b.Values.Wide, "a scalar value serves a wide draw")
		testkit.Equal(t, b.Values.Pin, "", "the key is an argument, not a field")
	})

	t.Run("a supplied reference retries the pin", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, kvStoreWith(t, "example.com/kv.Doc", "example.com/kv.Doc",
			[]storefixture.DirectiveOption{storefixture.KV(model.RefKey, "NewFake")},
			field{"ID", "string"}))
		testkit.True(t, b.Reference.Supplied(), "ref= replaced the derivation")
		testkit.Equal(t, b.Values.Pin, "ID", "but the pin derives on its own")
		testkit.True(t, b.Values.Wide, "so the wide pool still lands on pooled keys")
	})

	t.Run("a supplied reference with no derivable pin narrows", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, kvStoreWith(t, "example.com/kv.Doc", "example.com/kv.Doc",
			[]storefixture.DirectiveOption{storefixture.KV(model.RefKey, "NewFake")}))
		testkit.False(t, b.Values.Wide, "wide values keyed afresh never collide")
		testkit.Assert(t, b.Values.WhyNarrow).Contains("pin a wide draw",
			"and the header says what is missing")
	})
}

// TestValuePoolGuardsEveryDrawer holds the refusal that stands between a
// mistyped pool and the generator's worst output: code that does not compile.
//
// The values pool is one local of one type. Any action drawing from it hands
// that local to its method, so a second drawer taking another type produces a
// call no signature accepts — and nothing before `go build` says so, over a
// file the consumer did not write. Four shapes draw from the pool; before
// this guard covered them, three drew unchecked.
func TestValuePoolGuardsEveryDrawer(t *testing.T) {
	t.Parallel()

	// No reader, so the reference is the twin — which is the configuration
	// these two shapes ship in. Under a derived oracle they are held inert
	// before the guard is reached, so a fixture with a reader would prove
	// nothing about the guard: the corpus's own answeringwriter and mutator
	// fixtures both report under "model/twin" and both drive their action.
	t.Run("the answering writer and the mutator are measured too", func(t *testing.T) {
		t.Parallel()

		s := drawersOnly(t)
		b := bindingsOf(t, s)
		testkit.True(t, b.Reference.Twin(),
			"the twin is what lets these shapes drive at all")

		reasons := map[string]string{}
		for _, sk := range b.Skipped {
			reasons[sk.Method] = sk.Reason
		}
		testkit.Assert(t, reasons["Note"]).Contains("where the values pool draws",
			"an answering writer taking another type is refused, not driven")
		testkit.Assert(t, reasons["Bump"]).Contains("where the values pool draws",
			"and so is a mutator")

		driven := map[string]bool{}
		for _, a := range b.Actions {
			driven[a.Method] = true
		}
		testkit.True(t, driven["Store"], "the writer that types the pool still drives")
		testkit.False(t, driven["Note"], "neither mistyped drawer reaches the sequences")
		testkit.False(t, driven["Bump"], "neither mistyped drawer reaches the sequences")
	})

	// The baseline half: with a composite present, the pool takes *its* type,
	// so measuring against the plain writer inverts both verdicts — admitting
	// the drawer that mismatches and refusing the one that matches.
	t.Run("a composite writer supplies the baseline it types the pool with", func(t *testing.T) {
		t.Parallel()

		s := compositePoolWith(t)
		b := bindingsOf(t, s)

		testkit.Equal(t, b.Values.Q, "example.com/two.Body",
			"the composite types the pool, which is the premise of this test")

		reasons := map[string]string{}
		for _, sk := range b.Skipped {
			reasons[sk.Method] = sk.Reason
		}
		testkit.Assert(t, reasons["Stash"]).Contains("example.com/two.Tag",
			"the writer taking another type is refused, naming what it takes")
		testkit.Assert(t, reasons["Stash"]).Contains("example.com/two.Body",
			"and naming what the pool draws")

		driven := map[string]bool{}
		for _, a := range b.Actions {
			driven[a.Method] = true
		}
		testkit.True(t, driven["Save"],
			"the writer agreeing with the composite is driven, not refused for disagreeing with a writer")
		testkit.False(t, driven["Stash"], "and the disagreeing one never reaches the sequences")
	})

	// A method whose classification carried no value type at all still has to
	// read as a method problem. "takes  where the values pool draws X" reads
	// as a rendering bug and sends the reader to the generator, which is the
	// one place the fix is not.
	t.Run("an unstamped drawer names the absence rather than rendering a blank", func(t *testing.T) {
		t.Parallel()

		s := drawersOnly(t)
		// Cleared rather than never set: stampShape skips an empty value, and
		// the state being modelled is a detector that claimed the shape and
		// said nothing about the type it carries.
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Note" {
					shape.MetaValueType.Set(m.EnsureMeta(), "", "test")
				}
			}
		}
		b := bindingsOf(t, s)

		reasons := map[string]string{}
		for _, sk := range b.Skipped {
			reasons[sk.Method] = sk.Reason
		}
		testkit.Assert(t, reasons["Note"]).Contains("no stamped value type",
			"the refusal names the missing classification")
	})
}

// TestStickyRefinement walks the conflict the corpus surfaced the day the
// pools went wide: a sticky reader refines the oracle to its pinning form,
// and negates the observability law the writer's shape would otherwise earn —
// on a sticky store the two claims contradict at the first overwrite.
func TestStickyRefinement(t *testing.T) {
	t.Parallel()

	s := kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc", field{"ID", "string"})
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name == "Get" {
				shape.MetaMixins.Set(m.EnsureMeta(), []string{"sticky"}, "test")
			}
		}
	}
	b := bindingsOf(t, s)

	testkit.True(t, b.Reference.Pins, "the reader's claim refines the oracle")
	testkit.Equal(t, b.Reference.StoreType(), "StickyStore", "to its pinning form")

	unbound := map[string]string{}
	for _, u := range b.Unbound {
		unbound[u.Method] = u.Reason
	}
	testkit.Assert(t, unbound[lawid.WriteObservable]).Contains("sticky claim",
		"the negated law is listed with the contradiction, not silently absent")
	for _, l := range b.Laws {
		testkit.NotEqual(t, l.ID, lawid.WriteObservable, "and never bound")
	}
}

// TestGenSupply pins the gen= key: the consumer's generator constructor
// replaces reflection as the wide arm — outranking every narrowing verdict,
// because the consumer authored the domain — and a qualified spelling is
// refused the way ref='s is.
func TestGenSupply(t *testing.T) {
	t.Parallel()

	s := mixed(t, storefixture.KV(model.GenKey, "PayloadGen"))
	b := bindingsOf(t, s)
	testkit.Equal(t, b.Values.GenFunc, "PayloadGen", "the supply is recorded")
	testkit.True(t, b.Values.Wide, "and the pool goes wide through it")

	t.Run("a qualified constructor is refused", func(t *testing.T) {
		t.Parallel()
		s := mixed(t, storefixture.KV(model.GenKey, "other.PayloadGen"))
		got := generateBoth(t, s).Diagnostics()
		testkit.Equal(t, len(got), 1, "one diagnostic")
		testkit.Assert(t, got[0].Message).Contains("routed output package",
			"naming where the constructor must live")
	})
}

// TestDrainFixtures walks the writer-plus-collector fork both ways: a keyed
// value upserts and gets the map, a bare value appends and gets the
// collection — deduplicating exactly where the claim says so.
func TestDrainFixtures(t *testing.T) {
	t.Parallel()

	t.Run("a keyed value selects the map", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, drainStore(t, true))
		testkit.Equal(t, b.Reference.StoreType(), "MapStore",
			"an ID or Key field means upsert semantics")
		testkit.Equal(t, b.Reference.KeyField, "Key", "keyed on the conventional field")
		testkit.False(t, b.Reference.Collects(), "the map is not a collection")

		ops := map[string]string{}
		for _, am := range b.Adapter {
			ops[am.Sig.Name] = am.Op
		}
		testkit.Equal(t, ops["Items"], "Values", "the collector drains the map's values")

		testkit.True(t, b.Values.Wide, "the walk recurses through the nested struct")
		testkit.Equal(t, b.Values.Pin, "Key", "pinned on the upsert field")
		testkit.Equal(t, b.Keys.Field, b.Values.Field+".Key",
			"with no reader, the fixture values' own keys are the colliding set")
		testkit.True(t, b.UsesKeys(), "which the pin draws from")

		var bound *model.LawBinding
		for _, l := range b.Laws {
			if l.ID == lawid.StreamNoDuplicates {
				bound = l
			}
		}
		testkit.True(t, bound != nil, "the no-duplicates law binds")
		testkit.Equal(t, bound.Fields[0].Kind(), bound.Fields[0].KindName,
			"its drain renders through the field's template")
	})

	t.Run("a bare value selects the collection, deduplicating by claim", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, drainStore(t, false))
		testkit.True(t, b.Reference.Collects(), "no identity field, so append-and-drain")
		testkit.Equal(t, b.Reference.StoreType(), "SetCollection",
			"the no-duplicates claim refines the log into a set")
	})
}

// TestHistoryDrainForcesTheLog pins the drain fork's history arm: a claim
// whose vocabulary is repeats outranks the upsert inference an incidental
// Key field would trigger.
//
// Stamped overmatch rather than an isolation claim, which was the example
// until an isolation level became an admission policy: a store that refuses
// the operation an anomaly would need defeats every immediate oracle, so
// that claim never reaches the collection-or-map fork this pins. The drain
// claims still do — overmatch says a drain owes every write behind it, and
// a map inferred from a Key field would drop the second write to one key.
func TestHistoryDrainForcesTheLog(t *testing.T) {
	t.Parallel()

	s := drainStore(t, true)
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			// Replacing the fixture's noduplicates claim outright: a history
			// under a dedupe claim is a different fixture's question.
			shape.MetaMixins.Set(m.EnsureMeta(), []string{"overmatch"}, "test")
		}
	}
	b := bindingsOf(t, s)
	testkit.True(t, b.Reference.Collects(), "writes accumulate; they do not upsert")
	testkit.False(t, b.Reference.Dedupe, "and identical writes repeat")
}

// TestIsolationClaimDefeatsTheOracle pins the verdict that replaced the
// isolation fixtures' log reference: the level is a policy about what the
// store REFUSES, and every derived oracle records what it is handed.
//
// The corpus proved both halves. Left deriving a log, the reference admitted
// entries the subject refused and the differential reddened correct code;
// before that, with the subject recording passively, the anomaly laws found
// anomalies the drawn values had fabricated. The twin refuses alongside.
func TestIsolationClaimDefeatsTheOracle(t *testing.T) {
	t.Parallel()

	s := drainStore(t, true)
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			shape.MetaMixins.Set(m.EnsureMeta(), []string{"snapshotisolation"}, "test")
		}
	}
	b := bindingsOf(t, s)
	testkit.True(t, b.Reference.Twin(), "an admission policy leaves no oracle to derive")
	testkit.False(t, b.Reference.Collects(), "and the log form is not reached at all")
}
