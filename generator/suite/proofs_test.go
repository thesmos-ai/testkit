// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// Every defect variant this file claims to render has a template, and
// every template it names is a variant.
//
// The census the whole dispatch turns on. A kind in the map with no
// template fails the backend by name at render time — loud, but in a
// run rather than in a test — and a template no kind names is a body
// nothing reaches, which is silent. Held here rather than trusted.
func TestDefectRenderedNamesOnlyDeclaredVariants(t *testing.T) {
	t.Parallel()

	declared := map[projection.DefectKind]bool{}
	for _, k := range projection.DefectKinds() {
		declared[k] = true
	}

	for kind := range defectRendered() {
		testkit.True(t, declared[kind],
			"defectRendered names "+string(kind)+", which the projection does not declare")
	}
}

// A run pairs a defect with every check it stamps Proven, and names the
// rest.
//
// The two halves have to come out of one walk. prove.All fails a Proven
// claim with no evidence AND evidence for a claim nothing made, so a
// proofs file derived separately from the rows it is about would fail on
// whichever of the two drifted first.
func TestProofsOfPairsEveryProvenCheckAndNamesTheRest(t *testing.T) {
	t.Parallel()

	iface := provableIface()
	checks := checkEmitsOf(sdk.BaseEmit{}, iface, provableInventory(), true, false)
	testkit.Len(t, checks, 2, "both rows render a body")

	defects, unproven := proofsOf(sdk.BaseEmit{}, "example.com/pkg", nil, iface, checks)

	testkit.Len(t, defects, 2, "both rows carry a defect this generator spells")
	testkit.Equal(t, string(defects[0].Kind()), string(projection.KindStubPanic),
		"and the node's kind is its defect's template")
	testkit.Equal(t, defects[0].Group(), "Get", "keyed through the index member")
	testkit.Equal(t, defects[0].Accessor(), "Smoke", "and its accessor within it")
	testkit.Len(t, unproven, 0, "so nothing is left to name")
}

// A defect that must ANSWER is refused where the method's result admits
// no derivable value.
//
// The guard the whole per-check renderability split exists for: an
// echo-beside-error over an unsamplable result would return that
// result's ZERO, which is the very thing the check asks for — a defect
// asserting the claim it was built to break, and a proof that passes
// having proved the opposite. The row loses its stamp instead.
func TestProofsOfRefusesADefectItCannotGiveAValue(t *testing.T) {
	t.Parallel()

	// A func result: nothing the sampler can write a literal for.
	iface := Iface{
		Name:  "Store",
		Token: "store",
		Methods: []subject.Method{{Sig: &golang.Sig{
			Name: "Get",
			Returns: []golang.Return{
				{Local: "r0", Source: &node.TypeRef{Name: "Handler"}},
				{Local: "r1", Error: true},
			},
		}}},
	}
	inv := projection.Inventory{Checks: []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: "Get", Seg: vocab.SegZeroValue},
		Class:       vocab.ClassZeroValue,
		Body:        projection.ZeroOnCancel{Call: projection.CallPlan{Method: "Get"}},
		Falsifiable: vocab.Proven(),
		Defect:      projection.EchoBesideError{Option: projection.OptionName("Store", "Get")},
	}}}

	checks := checkEmitsOf(sdk.BaseEmit{}, iface, inv, true, false)
	testkit.Len(t, checks, 1, "the check still emits — only its evidence is missing")

	defects, unproven := proofsOf(sdk.BaseEmit{}, "example.com/pkg", nil, iface, checks)
	testkit.Len(t, defects, 0, "no defect is planted for a value nothing can spell")
	testkit.Equal(t, unproven, []string{"Get/zero-on-error"}, "and the row is named")
	testkit.False(t, checks[0].Proven, "with its stamp withdrawn rather than shipped")
}

// A defect's spelling comes from the method it overrides.
func TestDefectViewOfSpellsTheOverride(t *testing.T) {
	t.Parallel()

	iface := provableIface()
	got := defectViewOf("example.com/pkg", iface.Package, nil,
		iface.Name, iface.Methods[0], panicPlan())

	t.Run("names the double it plants through", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, got.Ctor, "NewStoreStub", "the constructor")
		testkit.Equal(t, got.Option, "WithStoreGet", "and the one-method override")
	})

	t.Run("words the defect as a sentence", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, got.DefectName, "a Store whose Get panics",
			"the report's subject line reads as prose")
	})

	t.Run("quotes its red by constant, not by text", func(t *testing.T) {
		t.Parallel()
		// The identifier, so rewording the primitive upstream breaks the
		// generated file's compile rather than the proof's evidence.
		testkit.Equal(t, got.ReasonConst, "RedPanicked",
			"the smoke family's failure is worded in the engine")
	})
}

// The article follows the interface's own first letter.
//
// These strings are read in failure output beside real ones, and "a
// AnsweringWriter" is the kind of wrongness that makes a reader distrust
// the line it sits in.
func TestDefectNameAgreesWithItsSubject(t *testing.T) {
	t.Parallel()

	type article struct{ iface, want string }
	testkit.TableTest(t, []article{
		{"Store", "a Store whose Get panics"},
		{"AnsweringWriter", "an AnsweringWriter whose Get panics"},
	}, func(t *testing.T, tc article) {
		testkit.Equal(t, projection.DefectName(tc.iface, "Get panics"), tc.want,
			"the article follows the name")
	})
}

// A variant with no clause still reads as a sentence.
//
// The fallback is its own slug, which is deliberate: it reads as a slug
// among prose, so the missing row is visible in the generated file rather
// than only in this table.
func TestDefectClauseFallsBackToTheVariantSlug(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, defectClause("Get", projection.FreshMedium{}), "Get fresh-medium",
		"an unworded variant names itself")
}

// An override's parameters are all blank and its results are all named
// or all anonymous, because Go's grammar forbids the mixture and the
// backend reports that as a render error rather than emitting it.
func TestOverrideSignaturesAreUniformlyNamed(t *testing.T) {
	t.Parallel()

	sig := &golang.Sig{
		Name:    "Get",
		Params:  []golang.Param{{Name: "ctx"}, {Name: "id"}},
		Returns: []golang.Return{{Local: "r0"}, {Local: "r1", Error: true}},
	}

	t.Run("every parameter is blank", func(t *testing.T) {
		t.Parallel()
		got := anonParams(sig)
		testkit.Len(t, got, 2, "one per parameter")
		for _, p := range got {
			testkit.Equal(t, p.Name, "_", "read by no defect body, so named by none")
		}
	})

	t.Run("a panicking body's results carry no names", func(t *testing.T) {
		t.Parallel()
		for _, r := range anonReturns(sig) {
			testkit.Equal(t, r.Name, "", "nothing returns, so nothing needs binding")
		}
	})

	t.Run("an answering body's results all carry one", func(t *testing.T) {
		t.Parallel()
		// This is what lets a bare `return` answer every slot's zero
		// without the generator spelling a zero of a type it may not be
		// able to name.
		got := namedReturns(sig)
		testkit.Equal(t, got[0].Name, "r0", "the value slot keeps the projection's own local")
		testkit.Equal(t, got[1].Name, errLocal, "and the error slot binds by a fixed name")
	})

	t.Run("a method answering nothing yields empty lists", func(t *testing.T) {
		t.Parallel()
		bare := &golang.Sig{Name: "Close"}
		testkit.Len(t, anonReturns(bare), 0, "no results to declare")
		testkit.Len(t, namedReturns(bare), 0, "and none to name")
	})
}

// Layout decides where the harness lands, and the companion follows it —
// down to every defect under it, which renders standalone and holds no
// path back to the file it sits in.
func TestSetOutputPackagesRepointsTheFileAndItsDefects(t *testing.T) {
	t.Parallel()

	p := &Proofs{
		Pkg: "example.com/provisional",
		Defects: &PlantedDefects{Rows: []*ProofEmit{
			{defectView: defectView{Pkg: "example.com/provisional"}},
		}},
	}

	t.Run("takes the primary output's package", func(t *testing.T) {
		t.Parallel()
		q := &Proofs{Pkg: "example.com/provisional", Defects: p.Defects}
		q.SetOutputPackages(map[string]string{"": "example.com/routed"})
		testkit.Equal(t, q.Pkg, "example.com/routed", "the file follows the harness")
		testkit.Equal(t, q.Defects.Rows[0].Pkg, "example.com/routed",
			"and so does every defect, which the backend renders on its own")
	})

	t.Run("keeps the provisional value where routing derived none", func(t *testing.T) {
		t.Parallel()
		// A wrong package is a compile error naming the symbol; a bare
		// name binds silently to whatever else is in scope.
		q := &Proofs{Pkg: "example.com/provisional"}
		q.SetOutputPackages(map[string]string{"test": "example.com/other"})
		testkit.Equal(t, q.Pkg, "example.com/provisional", "rather than blanking it")
	})
}

// provableIface is one interface with a method the smoke and
// zero-on-error families both reach.
func provableIface() Iface {
	return Iface{
		Name:  "Store",
		Token: "store",
		Methods: []subject.Method{{Sig: &golang.Sig{
			Name: "Get",
			Returns: []golang.Return{
				{Local: "r0", Source: &node.TypeRef{Name: "string"}},
				{Local: "r1", Error: true},
			},
		}}},
	}
}

func panicPlan() projection.CheckPlan {
	return projection.CheckPlan{
		ID:          projection.IDPlan{Method: "Get", Seg: vocab.SegSmoke},
		Class:       vocab.ClassSmoke,
		Body:        projection.SmokeSurvives{Call: projection.CallPlan{Method: "Get"}},
		Falsifiable: vocab.Proven(),
		Defect: projection.StubPanic{
			Clause: projection.Clause{Text: "Get panics"},
			Option: projection.OptionName("Store", "Get"),
		},
	}
}

// provableInventory holds one check whose defect this generator spells
// and one whose defect it does not, which is the split proofsOf reports.
func provableInventory() projection.Inventory {
	return projection.Inventory{Checks: []projection.CheckPlan{
		panicPlan(),
		{
			ID:          projection.IDPlan{Method: "Get", Seg: vocab.SegZeroValue},
			Class:       vocab.ClassZeroValue,
			Body:        projection.ZeroOnCancel{Call: projection.CallPlan{Method: "Get"}},
			Falsifiable: vocab.Proven(),
			Defect: projection.EchoBesideError{
				Option: projection.OptionName("Store", "Get"),
			},
		},
	}}
}

// A contributing tier plants through this file rather than building the
// defect itself.
//
// How a double is named, what an override is spelled at and which
// variants have templates are all this file's facts. A contributor that
// knew them would be a second place holding them, and the first sign of
// a disagreement would be a proofs map that does not compile.
func TestPlantRecordsAContributedDefect(t *testing.T) {
	t.Parallel()

	iface := provableIface()
	p := &Proofs{Pkg: "example.com/pkg", SourcePkg: iface.Package}
	p.IfaceName = iface.Name

	planted := p.Plant(nil, iface.Methods[0], panicPlan(), "Smoke")

	testkit.True(t, planted, "a variant with a template is written out")
	testkit.Len(t, p.Defects.Rows, 1, "and joins the map the run surface renders")
	testkit.Equal(t, p.Defects.Rows[0].Accessor(), "Smoke",
		"under the accessor the contributor named it by")
}

// A row with nothing planted for it is refused, so its stamp can move
// with it.
//
// The parity gate refuses a check claiming Proven with no defect beside
// it as firmly as the reverse, and it reports that against the generated
// package — where a reader would take it for a fault in their own code.
func TestPlantRefusesWhatItCannotWrite(t *testing.T) {
	t.Parallel()

	iface := provableIface()
	p := &Proofs{Pkg: "example.com/pkg", SourcePkg: iface.Package}
	p.IfaceName = iface.Name

	bare := panicPlan()
	bare.Defect = nil

	testkit.False(t, p.Plant(nil, iface.Methods[0], bare, "Smoke"),
		"a row carrying no defect plants nothing")
	testkit.True(t, p.Defects == nil,
		"and adds no entry the gate would then demand a claim for")
}
