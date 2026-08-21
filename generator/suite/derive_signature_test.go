// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
	"go.thesmos.sh/testkit/generator/suite"
)

// deriveCtx is the context parameter every ctx-taking fixture method
// shares; [golang.IsContext] answers from the qualified spelling.
func deriveCtx() golang.Param {
	return golang.Param{Name: "ctx", Source: storefixture.PkgNamed("context", "Context")}
}

// storeGet is the fully-armed shape: context, one draw, a named
// result beside the error — every signature family reaches it.
func storeGet() subject.Method {
	return subject.Method{
		Sig: &golang.Sig{
			Name:   "Get",
			Params: []golang.Param{deriveCtx(), keyParam("key")},
			Returns: []golang.Return{
				{Source: storefixture.Named("Value")},
				{Error: true},
			},
		},
		ArgFields: []string{"Key"},
	}
}

// storeIface pairs the methods with a fixture that can deliver every
// draw the fixtures above declare.
func storeIface(methods ...subject.Method) suite.Iface {
	return suite.Iface{
		Name: "Store", Token: "store", Qualifier: "store", Methods: methods,
		Fixture: subject.Fixture{Fields: []subject.FixtureField{{
			Name:   "Key",
			Sample: golang.Sample{Text: `"k"`},
			Other:  golang.Sample{Text: `"o"`},
		}}},
	}
}

// familyCase is one method shape and the ID set its rules license.
type familyCase struct {
	name  string
	iface suite.Iface
	want  []vocab.ID
}

func (c familyCase) Name() string { return c.name }

func TestSignatureDerivesTheFamilies(t *testing.T) {
	t.Parallel()

	closeM := subject.Method{Sig: &golang.Sig{
		Name:    "Close",
		Params:  []golang.Param{deriveCtx()},
		Returns: []golang.Return{{Error: true}},
	}}
	noCtx := subject.Method{Sig: &golang.Sig{
		Name:    "Len",
		Returns: []golang.Return{{Source: storefixture.Named("int")}, {Error: true}},
	}}
	noErr := subject.Method{
		Sig: &golang.Sig{
			Name:    "Peek",
			Params:  []golang.Param{deriveCtx(), keyParam("key")},
			Returns: []golang.Return{{Source: storefixture.Named("Value")}},
		},
		ArgFields: []string{"Key"},
	}

	testkit.TableTest(t, []familyCase{
		{
			"a fully-armed method reaches every family",
			storeIface(storeGet()),
			[]vocab.ID{"Get/smoke", "Get/cancel", "Get/nilcontext", "Get/deadline", "Get/zero-on-error"},
		},
		{
			"a teardown-shaped method never carries deadline",
			storeIface(closeM),
			[]vocab.ID{"Close/smoke", "Close/cancel", "Close/nilcontext"},
		},
		{
			"a context-less method carries no context families",
			storeIface(noCtx),
			[]vocab.ID{"Len/smoke"},
		},
		{
			"no error result, no context families and no zero",
			storeIface(noErr),
			// The engine primitives judge a `func(ctx) error`; a method
			// with no error channel has nothing to report a cancelled
			// context through, so the smoke is the whole family.
			[]vocab.ID{"Peek/smoke"},
		},
		{
			"declared totality excludes the zero family alone",
			storeIface(func() subject.Method {
				m := storeGet()
				m.Mixins = []string{suite.MixinTotal}
				return m
			}()),
			[]vocab.ID{"Get/smoke", "Get/cancel", "Get/nilcontext", "Get/deadline"},
		},
	}, func(t *testing.T, tc familyCase) {
		plans, refusals := suite.Signature{}.Derive(tc.iface)
		testkit.Len(t, refusals, 0, "derivable shapes refuse nothing")
		got := make([]vocab.ID, len(plans))
		for i, p := range plans {
			id, err := p.ID.Render()
			testkit.NoError(t, err, "the derived ID is well formed")
			got[i] = id
		}
		testkit.Equal(t, got, tc.want, "the rules license exactly these checks, in family order")
	})
}

func TestSignatureShapesTheChecks(t *testing.T) {
	t.Parallel()

	plans, _ := suite.Signature{}.Derive(storeIface(storeGet()))
	byID := map[vocab.ID]projection.CheckPlan{}
	for _, p := range plans {
		id, err := p.ID.Render()
		testkit.NoError(t, err, "the derived ID is well formed")
		byID[id] = p
	}
	wantCall := projection.CallPlan{
		Method: "Get",
		Args:   []projection.Expr{projection.ExprCtx, projection.FixtureCall(projection.ExprFixture, "Key")},
	}

	t.Run("the smoke survives with the fixture draw", func(t *testing.T) {
		t.Parallel()
		p := byID["Get/smoke"]
		testkit.Equal(t, p.Body, projection.Body(projection.SmokeSurvives{Call: wantCall}),
			"the smoke body carries the derived call")
		testkit.Equal(
			t,
			p.Defect,
			projection.Defect(projection.StubPanic{
				Clause: projection.Clause{Text: "Get panics"},
				Option: projection.OptionName("Store", "Get"),
			}),
			"the smoke is proven by the panicking double",
		)
		testkit.Equal(t, p.Class, vocab.ClassSmoke, "class buckets the report")
	})

	t.Run("nilcontext is proven by the accepting double", func(t *testing.T) {
		t.Parallel()
		p := byID["Get/nilcontext"]
		testkit.Equal(
			t,
			p.Defect,
			projection.Defect(projection.AnswersAnyway{
				Clause: projection.Clause{Text: "Get forgives a nil context and answers"},
				Option: projection.OptionName("Store", "Get"),
			}),
			"the claim's stronger arm — returns an error — needs the accepting defect",
		)
	})

	t.Run("the context families share the call and the swap defect", func(t *testing.T) {
		t.Parallel()
		for _, id := range []vocab.ID{"Get/cancel", "Get/deadline"} {
			p := byID[id]
			testkit.Equal(
				t,
				p.Defect,
				projection.Defect(projection.AnswersAnyway{
					Clause: projection.Clause{Text: "Get ignores the context it is handed"},
					Option: projection.OptionName("Store", "Get"),
				}),
				"a context family is proven by the context-ignoring double",
			)
		}
	})

	t.Run("every plan is Proven", func(t *testing.T) {
		t.Parallel()
		for id, p := range byID {
			testkit.Equal(t, p.Falsifiable.State, vocab.FalsifiableProven,
				"the signature families all carry planted defects: "+string(id))
		}
	})
}

func TestSignatureRefusesUnderivableDraws(t *testing.T) {
	t.Parallel()

	entry := subject.Method{
		Sig: &golang.Sig{
			Name:    "Append",
			Params:  []golang.Param{deriveCtx(), {Name: "e", Source: storefixture.Named("Entry")}},
			Returns: []golang.Return{{Error: true}},
		},
		ArgFields: []string{"Entry"},
	}
	iface := suite.Iface{Name: "Log", Token: "log", Qualifier: "log", Methods: []subject.Method{entry}}

	plans, refusals := suite.Signature{}.Derive(iface)
	testkit.Len(t, plans, 1, "the smoke survives an underivable draw — it needs a value, not a chosen one")
	id, err := plans[0].ID.Render()
	testkit.NoError(t, err, "the surviving plan renders its ID")
	testkit.Equal(t, id, vocab.ID("Append/smoke"), "and it is the smoke")
	testkit.Len(t, refusals, 1, "every family that compares an answer to an input folds into one refusal")
	testkit.Equal(t, refusals[0].What, "Append's judging signature checks",
		"the refusal names the families it covers, which is no longer all of them")
	testkit.Contains(t, refusals[0].Why, "Entry", "the refusal names the draw nothing supplies")
	testkit.Contains(t, refusals[0].Remedy, "LogConfig", "the remedy names where the value comes from")
	testkit.Contains(t, refusals[0].Remedy, "LogChecks", "and where the claim goes")
}

func TestSignaturePlansSatisfyTheInventory(t *testing.T) {
	t.Parallel()

	plans, _ := suite.Signature{}.Derive(storeIface(storeGet()))
	inv := projection.Inventory{Iface: "Store", Token: "store", Checks: plans}
	testkit.NoError(t, inv.Verify(), "derived plans hold the inventory's parity rules by construction")
}
