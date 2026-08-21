// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

func TestBodyKinds(t *testing.T) {
	t.Parallel()

	t.Run("unique", func(t *testing.T) {
		t.Parallel()
		seen := map[projection.BodyKind]bool{}
		for _, k := range projection.BodyKinds() {
			testkit.False(t, seen[k], "body kind "+string(k)+" must register once")
			seen[k] = true
		}
	})

	t.Run("dispatch-prefixed", func(t *testing.T) {
		t.Parallel()
		for _, k := range projection.BodyKinds() {
			testkit.HasPrefix(
				t,
				string(k),
				projection.BodyKindPrefix,
				"kinds are template names in the dispatch namespace",
			)
		}
	})

	// The budget counts EMITTED SHAPES, which is what it has to count to
	// mean anything. A kind named for the classification that asked for
	// it grows with the registry and the guard fires on arithmetic; a
	// kind named for the statements it emits is shared, and the guard
	// fires only when a rule genuinely wants code nothing else emits.
	// The set is named on the second footing.
	t.Run("within the design budget", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(projection.BodyKinds()) <= 19,
			"a new body shape is a design event, not a workaround — the budget is the guard; "+
				"before raising it, check the shape is not one an existing kind already emits")
	})
}

// simKindCase ties one sim kind to the runtime segment it must equal.
type simKindCase struct {
	name string
	kind projection.SimKind
	seg  string
}

func (c simKindCase) Name() string { return c.name }

// strengthCase ties one body variant to the class of evidence its
// generated statements gather.
type strengthCase struct {
	name string
	body projection.Body
	want suite.Strength
}

func (c strengthCase) Name() string { return c.name }

// TestBodyStrength pins what each shape looks at before it passes.
//
// The table is written out rather than derived, because deriving it
// from the method would assert nothing: the point is that a reader
// who disagrees with a row has somewhere to argue. The completeness
// subtest is what stops the table going stale — a variant added to
// BodyKinds without a row here fails, which is the moment the
// decision is cheapest to make.
//
//nolint:thelper // the case body is the test, not a helper; see core/lawid
func TestBodyStrength(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, bodyStrengths(), func(t *testing.T, tc strengthCase) {
		testkit.Equal(t, tc.body.Strength(), tc.want,
			"a body's strength is what the run reports about its own reach; "+
				"overstating it hides a check that only ever asked whether the call returned")
	})

	t.Run("covers every kind", func(t *testing.T) {
		t.Parallel()
		seen := map[projection.BodyKind]bool{}
		for _, tc := range bodyStrengths() {
			seen[tc.body.BodyKind()] = true
		}
		for _, k := range projection.BodyKinds() {
			testkit.True(t, seen[k],
				"body kind "+string(k)+" has no strength row; decide what it looks at")
		}
	})
}

// bodyStrengths is the table, as a function so both subtests read the
// same rows.
func bodyStrengths() []strengthCase {
	return []strengthCase{
		{"smoke survives", projection.SmokeSurvives{}, suite.StrengthErrorOnly},
		{"guarded call", projection.GuardedCall{}, suite.StrengthErrorOnly},
		{"repeat probe", projection.RepeatProbe{}, suite.StrengthErrorOnly},
		{"reports sentinel", projection.ReportsSentinel{}, suite.StrengthErrorOnly},
		{"partner agrees", projection.PartnerAgrees{}, suite.StrengthErrorOnly},
		{"zero on miss", projection.ZeroOnMiss{}, suite.StrengthObserved},
		{"zero on cancel", projection.ZeroOnCancel{}, suite.StrengthObserved},
		{"answers zero", projection.AnswersZero{}, suite.StrengthObserved},
		{"hit probe", projection.HitProbe{}, suite.StrengthObserved},
		{"count probe", projection.CountProbe{}, suite.StrengthObserved},
		{"hook fires", projection.HookFires{}, suite.StrengthObserved},
		{"non-zero answer", projection.NonZeroAnswer{}, suite.StrengthObserved},
		{"read either side of an act", projection.ReadActRead{}, suite.StrengthObserved},
		{"read after two writes", projection.WriteWriteRead{}, suite.StrengthObserved},
		{"law leg", projection.LawLeg{}, suite.StrengthDifferential},
		{"differential leg", projection.DifferentialLeg{}, suite.StrengthDifferential},
		{"sim leg", projection.SimLeg{}, suite.StrengthDifferential},
	}
}

//nolint:thelper // the case body is the test, not a helper; see core/lawid
func TestSimKindsAreTheRuntimeVocabulary(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []simKindCase{
		{"recovery", projection.SimRecovery, suite.SegRecovery},
		{"crash", projection.SimCrash, suite.SegCrash},
		{"fault", projection.SimFault, suite.SegFault},
	}, func(t *testing.T, tc simKindCase) {
		testkit.Equal(t, string(tc.kind), tc.seg,
			"the sim vocabulary has one home; a drifted spelling mints IDs the runtime refuses")
	})
}
