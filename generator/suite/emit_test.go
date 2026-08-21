// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal for the reason derive_laws_test.go is: the pairing reads the
// projections this package builds, and the view it produces is its own.
package suite

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// bindCase is one signature and the two bindings it spells.
type bindCase struct {
	name    string
	returns []golang.Return
	discard string
	errBind string
}

func (c bindCase) Name() string { return c.name }

// The bindings are the signature's, spelled the way the packs spell
// them.
//
// Both rules are one-liners and both are load-bearing: a discard short
// by one blank does not compile, and an error bound where the packs
// return directly is a local the reader has to follow to the next line.
func TestViewBindsWhatTheSignatureReturns(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []bindCase{
		{
			"an error alone is returned, never bound",
			[]golang.Return{{Error: true}},
			"_ =", "",
		},
		{
			"a value beside an error is bound past one blank",
			[]golang.Return{{Source: storefixture.Named("Value")}, {Error: true}},
			"_, _ =", "_, err :=",
		},
		{
			"two values beside an error take two blanks",
			[]golang.Return{
				{Source: storefixture.Named("Value")},
				{Source: storefixture.Named("bool")},
				{Error: true},
			},
			"_, _, _ =", "_, _, err :=",
		},
		{
			"a method answering nothing discards nothing",
			nil,
			"", "",
		},
	}, func(t *testing.T, tc bindCase) {
		m := subject.Method{Sig: &golang.Sig{Name: "Get", Returns: tc.returns}}
		got := viewOf(Iface{Name: "Store", Token: "store"}, m)
		testkit.Equal(t, got.Discard, tc.discard, "the discard covers every result")
		testkit.Equal(t, got.ErrBind, tc.errBind, tc.name)
	})
}

// A check is paired with the method it is about, and one naming a
// method the interface does not declare is dropped.
//
// The drop matters more than the pairing: such a plan can only come
// from a deriver naming something that is not there, and emitting a
// call to it would fail in the consumer's build rather than in the run
// that produced it. Family-scoped plans carry no body of ours and are
// not paired at all.
func TestCheckEmitsPairWithTheirMethod(t *testing.T) {
	t.Parallel()

	iface := Iface{
		Name: "Store", Token: "store", Qualifier: "store",
		Methods: []subject.Method{{Sig: &golang.Sig{Name: "Get", Returns: []golang.Return{{Error: true}}}}},
	}
	inv := projection.Inventory{
		Iface: "Store", Token: "store",
		Checks: []projection.CheckPlan{
			{
				ID:    projection.IDPlan{Method: "Get", Seg: vocab.SegSmoke},
				Class: vocab.ClassSmoke,
				Body:  projection.SmokeSurvives{},
			},
			{
				ID:    projection.IDPlan{Method: "Gone", Seg: vocab.SegSmoke},
				Class: vocab.ClassSmoke,
				Body:  projection.SmokeSurvives{},
			},
			{
				ID:    projection.IDPlan{Family: vocab.FamilyModel, Qualifier: "store", Seg: vocab.SegLaws},
				Class: vocab.ClassLaws,
				Body:  projection.LawLeg{},
			},
		},
	}

	got := checkEmitsOf(sdk.BaseEmit{}, iface, inv, true, false)
	testkit.Len(t, got, 1, "the declared method pairs; the absent one and the family scope do not")
	testkit.Equal(t, got[0].Check, "storeGet", "and the check names the method through its constant")
	testkit.Equal(t, got[0].Recv, "s", "called through the subject's own initial")
	testkit.Equal(t, string(got[0].Kind()), string(projection.KindSmokeSurvives),
		"the node's kind is its body's template")
	testkit.Equal(t, got[0].AssertName(), "storeAssertGetSmoke",
		"and it carries the assertion the row names")
	testkit.Equal(t, got[0].ClassConst(), "ClassSmoke",
		"with the class named rather than spelled as a slug")
}

// A generic subject's rows claim Argued whatever their deriver stamped.
//
// The companion cannot instantiate a planted defect at concrete types —
// a Go test function takes no type parameters — so a Proven stamp there
// would be a claim with nothing behind it, and nothing would catch it:
// the parity gate only runs where a companion emitted a test.
func TestCheckEmitsArgueWhereNothingCanBePlanted(t *testing.T) {
	t.Parallel()

	iface := Iface{
		Name:  "Store",
		Token: "store",
		Methods: []subject.Method{
			{Sig: &golang.Sig{Name: "Get", Returns: []golang.Return{{Error: true}}}},
		},
	}
	inv := projection.Inventory{
		Checks: []projection.CheckPlan{
			{
				ID:          projection.IDPlan{Method: "Get", Seg: vocab.SegSmoke},
				Class:       vocab.ClassSmoke,
				Body:        projection.SmokeSurvives{Call: projection.CallPlan{Method: "Get"}},
				Falsifiable: vocab.Proven(),
				Defect:      projection.StubPanic{Option: projection.OptionName("Store", "Get")},
			},
		},
	}

	concrete := checkEmitsOf(sdk.BaseEmit{}, iface, inv, true, false)
	testkit.Len(t, concrete, 1, "the row emits either way")
	testkit.True(t, concrete[0].Proven,
		"a concrete subject's smoke check has a defect this generator spells")

	generic := checkEmitsOf(sdk.BaseEmit{}, iface, inv, false, false)
	testkit.False(t, generic[0].Proven,
		"the same check claims nothing where no companion can drive it")
	testkit.Assert(t, generic[0].Argument()).
		Contains("cannot name them", "and says which fact stopped it")
}
