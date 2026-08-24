// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal on purpose, the suite_internal_test.go precedent: the
// sentinel rule reads mixin params through the unexported mixinParams
// projection, which only [mixinParamsOf] over a stamped bag populates
// — the exported surface reaches it solely through the sdk pipeline.
package suite

import (
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/aggregator"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/reader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/writer"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"

	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// stampMethod builds one key-drawing method with a detected shape and
// attached mixins, its stamps set through the real shape keys.
func stampMethod(name, detected string, mixins ...string) subject.Method {
	src := &node.Method{Name: name}
	if detected != "" {
		shape.MetaShape.Set(src.EnsureMeta(), detected, "test")
	}
	return subject.Method{
		Sig: &golang.Sig{
			Name:   name,
			Params: []golang.Param{{Name: "key", Source: storefixture.Named("Key")}},
			Source: src,
		},
		Mixins:    mixins,
		ArgFields: []string{"Key"},
	}
}

// bareMethod is stampMethod without a draw, the teardown and
// aggregator shapes.
func bareMethod(name, detected string, mixins ...string) subject.Method {
	m := stampMethod(name, detected, mixins...)
	m.Params = nil
	m.ArgFields = nil
	return m
}

// sentinelReader is the kv Get shape: a reader declaring its OWN miss
// sentinel, read through the real param key.
func sentinelReader() subject.Method {
	m := stampReader("Get", MixinNotFound)
	// Stamped on the DECLARATION, which is where the annotator writes
	// it, and the projected map derived from that — the order the
	// pipeline runs in. Setting a detached bag and assigning the map
	// from it left the declaration bare, so a reader consulting the
	// source saw an unstamped method and the fixture pinned a claim no
	// real run produces.
	shape.MixinParamKey(MixinNotFound, MixinNotFoundSentinel).
		Set(m.Source.EnsureMeta(), "kv.ErrNotFound", "test")
	m.MixinParams = mixinParamsOf(m.Source.Meta(), m.Mixins)
	return m
}

// stampReader is stampMethod for a reader: the Value it answers beside
// the error.
//
// Its own helper rather than a return on every stamped method, because
// the mixin rules read the same builder — and an error channel they were
// asserting the absence of changes which gap they report.
func stampReader(name string, mixins ...string) subject.Method {
	m := stampMethod(name, reader.Name, mixins...)
	m.Returns = []golang.Return{{Source: storefixture.Named("Value")}, {Error: true}}
	return m
}

// stampWriter is stampMethod for a writer: the key beside the value it
// stores.
//
// A writer taking only a key stores nothing, which is not a writer — and
// the miss rule reads exactly that, asking whether some method here
// writes the kind of value the reader answers.
func stampWriter(name string) subject.Method {
	m := stampMethod(name, writer.Name)
	m.Params = append(m.Params, golang.Param{Name: "value", Source: storefixture.Named("Value")})
	m.ArgFields = append(m.ArgFields, "Value")
	return m
}

// stampIface pairs the methods with a fixture that can deliver the
// key draw.
func stampIface(methods ...subject.Method) Iface {
	return Iface{
		Name: "Store", Token: "store", Qualifier: "store", Methods: methods,
		Fixture: subject.Fixture{Fields: []subject.FixtureField{{
			Name:   "Key",
			Sample: golang.Sample{Text: `"k"`},
			Other:  golang.Sample{Text: `"o"`},
		}}},
	}
}

// writerIface is [stampIface] with the value a writer stores, for the
// cases that pair a reader with one.
//
// Its own helper rather than a field on the shared fixture: the mixin
// rules read the same fixture, and a Value field they did not ask for
// changes which gap they report first.
func writerIface(methods ...subject.Method) Iface {
	f := stampIface(methods...)
	f.Fixture.Fields = append(f.Fixture.Fields, subject.FixtureField{
		Name:   "Value",
		Sample: golang.Sample{Text: `"v"`},
		Other:  golang.Sample{Text: `"w"`},
	})
	return f
}

// seededIface is [stampIface] for a run that zips a corpus from its
// pools, which is what puts something there for a hit to find and
// leaves one key out for a miss to draw.
func seededIface(methods ...subject.Method) Iface {
	f := stampIface(methods...)
	f.Corpus = true
	return f
}

// stampCase is one stamp shape and the ID set its rule licenses,
// beside the number of gaps the rule names rather than emitting.
type stampCase struct {
	name     string
	iface    Iface
	want     []vocab.ID
	refusals int
}

func (c stampCase) Name() string { return c.name }

func TestStampsDeriveTheStampFamilies(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []stampCase{
		{
			// One refusal beside the row, and it is not a gap in the
			// rule: idempotent binds two laws as well, so the header
			// has to say that the repeat was tried once here and that
			// "N calls equal one" is the model tier's. See
			// TestATabledMixinNamesItsOtherTiersObligation.
			"idempotent probes the repeat, and names the rest",
			stampIface(bareMethod("Close", "", MixinIdempotent)),
			[]vocab.ID{"Close/idempotent"},
			1,
		},
		{
			"a stamped sentinel reader derives its miss",
			writerIface(sentinelReader(), stampWriter("Put")),
			[]vocab.ID{"Get/miss"},
			0,
		},
		{
			"a reader beside a writer derives its miss",
			writerIface(stampReader("Lookup"), stampWriter("Put")),
			[]vocab.ID{"Lookup/miss"},
			0,
		},
		{
			"a seeded reader derives miss and hit",
			seededIface(stampReader("Lookup")),
			[]vocab.ID{"Lookup/miss", "Lookup/hit"},
			0,
		},
		{
			"a seeded aggregator derives its count",
			seededIface(bareMethod("Size", aggregator.Name)),
			[]vocab.ID{"Size/count"},
			0,
		},
		{
			"an aggregator beside a writer licenses nothing",
			writerIface(bareMethod("Len", aggregator.Name), stampWriter("Put")),
			nil,
			0,
		},
		{
			// A transform: reader-shaped, but nothing writes and no
			// corpus seeds, so no draw is one nothing supplied.
			"a reader shape with nothing to supply it refuses",
			stampIface(stampReader("Encode")),
			nil,
			1,
		},
	}, func(t *testing.T, tc stampCase) {
		plans, refusals := Stamps{}.Derive(tc.iface)
		testkit.Len(t, refusals, tc.refusals, "the rule names exactly these gaps")
		got := make([]vocab.ID, 0, len(plans))
		for _, p := range plans {
			id, err := p.ID.Render()
			testkit.NoError(t, err, "the derived ID is well formed")
			got = append(got, id)
		}
		if len(tc.want) == 0 {
			testkit.Len(t, got, 0, "the rule licenses nothing here")
			return
		}
		testkit.Equal(t, got, tc.want, "the rule licenses exactly these checks")
	})
}

func TestStampsHoldTheCensusPosture(t *testing.T) {
	t.Parallel()

	t.Run("a law-backed stamp is the model tier's, and said so", func(t *testing.T) {
		t.Parallel()
		plans, refusals := Stamps{}.Derive(stampIface(stampMethod("Put", "", MixinTTL)))
		testkit.Len(t, plans, 0, "the laws deriver owns it")

		// Named rather than silent. It was silent, and the generated
		// header then listed a method's signature checks and never
		// mentioned the directive the consumer had written above it —
		// which reads as a file that checked it.
		testkit.Len(t, refusals, 1, "the directive is accounted for, not passed over")
		testkit.True(t, refusals[0].Elsewhere,
			"owned by another tier, which is not the same as a gap")
		testkit.False(t, refusals[0].Unaccounted,
			"and not a classification nobody has decided about")
	})

	t.Run("an unknown stamp refuses and marks itself unaccounted", func(t *testing.T) {
		t.Parallel()
		plans, refusals := Stamps{}.Derive(stampIface(stampMethod("Put", "", "brand-new-shape")))
		testkit.Len(t, plans, 0, "nothing derives from an unknown stamp")
		testkit.Len(t, refusals, 1, "the gap is named, never silent")
		testkit.Contains(t, refusals[0].What, "brand-new-shape", "the refusal names the stamp")

		// The flag rather than a phrase in the prose. The remedy is
		// consumer-facing and its wording is free to improve; what the
		// census reads is this field, and pinning the sentence instead
		// would fail the day someone reworded it without changing what
		// it means.
		testkit.True(t, refusals[0].Unaccounted,
			"an unknown stamp is the gap itself, not an argument for one")
		testkit.Equal(t, refusals[0].Licensed.Name, "brand-new-shape",
			"and the census can attribute it")
	})

	t.Run("an undeliverable draw refuses the stamp checks", func(t *testing.T) {
		t.Parallel()
		iface := Iface{
			Name:      "Store",
			Token:     "store",
			Qualifier: "store",
			Methods:   []subject.Method{stampMethod("Get", reader.Name)},
		}
		plans, refusals := Stamps{}.Derive(iface)
		testkit.Len(t, plans, 0, "no check derives over a draw nothing supplies")
		testkit.Len(t, refusals, 1, "the whole stamp set folds into one refusal")
		testkit.Equal(t, refusals[0].What, "Get's stamp checks", "the refusal names the method's stamp set")
	})

	t.Run("the corpus claims come out verbatim", func(t *testing.T) {
		t.Parallel()
		plans, _ := Stamps{}.Derive(writerIface(sentinelReader(), stampWriter("Put")))
		testkit.Len(t, plans, 1, "one miss check")
		testkit.Equal(t, plans[0].Claim, "Get reports ErrNotFound for a key nothing wrote",
			"the manifest spelling: bare sentinel, writer-fed verb")
	})
}

// The census gate, the tiers/actions pattern reused: every upstream
// classification is tabled here, law-backed in tiers, or recorded
// below with the reason it owes no rule row — an input another
// derivation consumes, a behaviour another tier owns, or a PENDING
// gap naming the work that closes it. Held equal to the live
// registries in both directions, so an eidos addition fails by name
// in the build that makes it stampable, and a recorded entry goes
// stale loudly the moment a row or law covers it. The names here are
// census data, not vocabulary: a misspelled key is an orphan the gate
// rejects.
//
// Three states, and the difference between them is who owns the work.
// A permanent placement says why the classification owes this tier no
// row at all. A MODEL TIER entry says the obligation is real and
// belongs to a tier that is not built yet, so the gate that closes it
// is the model generator rather than anything here. A PENDING entry is
// this tier's own work, undone — and must be empty by the flip, because
// deferred work with no owner is how a census stops meaning anything.

var recordedMixins = map[string]string{
	"concurrent": "the laws deriver lowers it to the linearizable leg. No suite row beside " +
		"it: two goroutines and no panic is observable only under -race, which the default " +
		"gate does not run, and a check asserting nothing there reads as coverage",
	"wrappedvia": "MODEL TIER: fn= names the wrapper reference rather than an operation " +
		"of the subject's own, and reaching the claim needs an input the callable FAILS " +
		"for. Induced failure is the model tier's machinery, so the model generator binds " +
		"it and no suite row is owed",
	"concurrentreaders": "MODEL TIER: concurrent readers need goroutines and a race " +
		"detector, neither of which a caller has — the model generator binds it",
	"retrysucceeds": "MODEL TIER: convergence under retry is a property over generated " +
		"sequences, not a fixed probe — attempts= declares the bound the property needs, " +
		"and the model generator binds it",
	"sample": "an input to the smoke rather than a claim of its own: builder= names " +
		"where a member of an unenumerable input space comes from, and builtSmoke borrows " +
		"from it instead of drawing a literal the subject may have no reason to accept. " +
		"The bare form states what this tier does by construction — it draws one value per " +
		"role and has no exhaustive mode to be told not to use",
	"deprecated": "documentation stamp: colours generated prose, owes no check",
	"notfound": "an identity, not a claim: it names WHAT a miss reports, and the check " +
		"that a miss IS reported is the reader shape's own — MissSentinel reads it, " +
		"which is what turns that check's body from the zero arm into the sentinel arm",
	"integrationonly": "run gate: scopes checks behind the integration env, owes none of its own",
	"scope": "documentation stamp, by upstream ruling: name= says what an axis MEANS and " +
		"licenses no check. The isolation it describes is partition's, which names the " +
		"observer as well — the two compose on one callable, naming form beside " +
		"checkable form",
	"errors": "documentation stamp, by upstream ruling: it marks the error returns as " +
		"contract rather than \"shouldn't happen\" and licenses nothing falsifiable. " +
		"Which sentinel answers which condition is notfound sentinel= and its siblings",
}

var recordedDetectors = map[string]string{
	"streamconsumer": "the shape exists to STOP a derivation rather than license one. Its " +
		"parameter is an interface no fixture can construct, and the detector's own reason " +
		"for being is that letting such a callable fall to reader produced checks a drained " +
		"stream passed vacuously. The signature families still cover the method, and the " +
		"undeliverable-draw refusal names the pool a consumer would supply",
	"closer":        "teardown shape: the signature families cover it, the after-close laws bind the rest",
	"voidlifecycle": "teardown shape: the signature families cover it, the after-close laws bind the rest",
	"mutator":       "a writer answering nothing: excluded from seeding on purpose, and the signature families cover it",
}

var recordedContracts = map[string]string{
	"circuit-breaker": "MODEL TIER: the protocol only shows itself under induced failure, " +
		"which a caller cannot cause — the model generator binds it",
	"leader-election": "MODEL TIER: a multi-node protocol needs a second subject, which " +
		"one harness does not have — the model generator binds it",
	"rate-limit": "MODEL TIER: a budget over time needs a clock the suite tier does not " +
		"control — the model generator binds it",
}

// assertCensus holds one axis's registry to the three-way partition.
func assertCensus(t *testing.T, registry []string, tabled map[string]stampRule, recorded map[string]string) {
	t.Helper()

	var uncovered []string
	for _, name := range registry {
		if _, ok := tabled[name]; ok {
			continue
		}
		if len(tiers.LawsFor(name)) > 0 {
			continue
		}
		if _, ok := recorded[name]; ok {
			continue
		}
		uncovered = append(uncovered, name)
	}
	slices.Sort(uncovered)
	testkit.Len(t, uncovered, 0, "every classification is tabled, law-backed, or recorded with a reason — uncovered: "+
		strings.Join(uncovered, ", "))

	var stale, orphaned []string
	for name := range recorded {
		if _, ok := tabled[name]; ok || len(tiers.LawsFor(name)) > 0 {
			stale = append(stale, name)
		}
		if !slices.Contains(registry, name) {
			orphaned = append(orphaned, name)
		}
	}
	slices.Sort(stale)
	slices.Sort(orphaned)
	testkit.Len(t, stale, 0, "a recorded entry a row or law now covers must be deleted: "+
		strings.Join(stale, ", "))
	testkit.Len(t, orphaned, 0, "a recorded entry the registry no longer carries is a typo or a removal: "+
		strings.Join(orphaned, ", "))
}

func TestStampCensusCoversTheMixinRegistry(t *testing.T) {
	t.Parallel()
	names := make([]string, 0, len(mixins.All()))
	for _, m := range mixins.All() {
		names = append(names, m.Name)
	}
	assertCensus(t, names, mixinRules(), recordedMixins)
}

func TestStampCensusCoversTheDetectorRegistry(t *testing.T) {
	t.Parallel()
	names := make([]string, 0, len(detectors.All()))
	for _, d := range detectors.All() {
		names = append(names, d.Name)
	}
	assertCensus(t, names, detectorRules(), recordedDetectors)
}

func TestStampCensusCoversTheContractRegistry(t *testing.T) {
	t.Parallel()
	names := make([]string, 0, len(contracts.All()))
	for _, c := range contracts.All() {
		names = append(names, c.Name)
	}
	// The contracts deriver keys its table on a (contract, role) pair,
	// because a rule written against the wrong member of a protocol
	// calls the wrong method. The census asks only which contracts are
	// tabled, so the roles are dropped here rather than widening the
	// shared gate to a shape only one axis has.
	tabled := make(map[string]stampRule, len(contractRules()))
	for _, e := range contractRules() {
		tabled[e.contract] = nil
	}
	assertCensus(t, names, tabled, recordedContracts)
}

// A classification the suite tier covers AND a law backs is named in
// both, which is what ADR-0028 changed.
//
// Under ADR-0018 the two arms were alternatives: a tabled rule won and
// the law-backed note was never reached, so a header listed `idempotent`
// among its checks and told the reader nothing about what was left. The
// row settles one call repeated; the laws settle N, for any N. A reader
// who takes the first for the second has been misled by an omission.
func TestATabledMixinNamesItsOtherTiersObligation(t *testing.T) {
	t.Parallel()

	plans, refusals := Stamps{}.Derive(
		stampIface(bareMethod("Close", "", MixinIdempotent)),
	)

	testkit.Len(t, plans, 1, "the suite row is still emitted")
	testkit.Len(t, refusals, 1, "and the obligation it does not reach is named beside it")

	r := refusals[0]
	testkit.True(t, r.Elsewhere, "it is another tier's, not a gap")
	testkit.Equal(t, r.Obligation, tiers.ObUniversal,
		"the obligation is named by what it needs, not by the classification")
	testkit.Contains(t, r.What, MixinIdempotent, "and the line names the directive they wrote")
	testkit.NotContains(t, r.What, string(tiers.ObUniversal),
		"but not the obligation's own word, which is ours and not a consumer's")
	testkit.Contains(t, r.Remedy, string(tiers.TierModel), "with the tier that holds it")
	testkit.Equal(t, r.Licensed.Name, MixinIdempotent,
		"licensed to the classification, so the census reads it as accounted for")
}

// A classification the suite tier covers alone gets no note, because
// there is nothing elsewhere to name.
func TestASuiteOnlyMixinNamesNothingElsewhere(t *testing.T) {
	t.Parallel()

	// sideeffect carries no law: observe, call, observe is the whole
	// claim, and a note pointing at a tier that checks nothing would be
	// worse than silence.
	_, refusals := Stamps{}.Derive(
		seededIface(stampMethod("Put", writer.Name, MixinSideEffect)),
	)

	for _, r := range refusals {
		testkit.False(t, r.Elsewhere && r.Licensed.Name == MixinSideEffect,
			"sideeffect binds no law, so no obligation of it is another tier's")
	}
}
