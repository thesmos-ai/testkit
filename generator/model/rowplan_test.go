// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/model"
)

// The rows this tier plans, and what decides each.
//
// Planned here rather than by the harness generator, which planned them
// while the harness's capability fields were projected from the plans: a
// clocked law's row was what opened the clock, so the row and the field
// had to be worked out together. This tier contributes the field itself
// now, so it can plan the row that needs it.
//
// Two families, not one. Every row that judges the subject against
// something modelling it reports under model; the crash row reports
// under sim, because it judges the subject against its own
// acknowledgements across a seam nothing else in the run crosses, and a
// report grouping the two would put a lost write beside a disagreeing
// reference.
func TestPlanRowsFollowsWhatBound(t *testing.T) {
	t.Parallel()

	b := bindingsOf(t, mixed(t))
	rows := model.PlanRows(b)

	testkit.True(t, len(rows) > 0, "an interface binding laws owes rows")
	for _, r := range rows {
		testkit.Contains(t, []string{vocab.FamilyModel, vocab.FamilySim}, r.ID.Family,
			"every row this tier plans reports under one of the two families it owns")
		testkit.NotEqual(t, r.Claim, "", "and states a claim a reader can disagree with")
		testkit.True(t, r.Body != nil,
			"and carries a body kind, or nothing can render it")
	}
}

// A law the catalogue words gets a row of its own, under its own
// identity and its own claim.
//
// The wording is the criterion, because a row exists to say one thing a
// reader can disagree with. A law with no sentence of its own has
// nothing to put in the Claim column but the bundle's, and N laws under
// one sentence cannot tell a reader which of them broke.
func TestPlanRowsGiveEachWordedLawItsRow(t *testing.T) {
	t.Parallel()

	b := bindingsOf(t, mixed(t))
	rows := model.PlanRows(b)

	byLaw := map[string]projection.CheckPlan{}
	for _, r := range rows {
		byLaw[r.ID.Seg] = r
	}
	for _, l := range b.RowedLaws() {
		row, planned := byLaw[l.ID]
		if !planned {
			// Refused, and the header says why — the alternative is a row
			// stating a sentence this package invented for somebody
			// else's law.
			continue
		}
		testkit.NotEqual(t, row.Claim, "",
			"the law's own sentence, from the catalogue that defines it")
		testkit.Len(t, row.Binds, 1, "and the law it reports under")
		testkit.Equal(t, row.Binds[0].Law, l.ID, "which is this one")
	}
}

// The differential row exists exactly where there is something to
// compare against.
func TestPlanRowsNeedAReference(t *testing.T) {
	t.Parallel()

	rows := model.PlanRows(bindingsOf(t, mixed(t)))

	var differential int
	for _, r := range rows {
		if r.ID.Seg == vocab.SegDifferential {
			differential++
		}
	}
	testkit.Equal(t, differential, 1,
		"one differential row where a reference is derived, and one only")
}

// A row's identity is composed the way the index composes it, because
// the index names the row and the row reports under the name.
func TestPlanRowsQualifyAsTheIndexDoes(t *testing.T) {
	t.Parallel()

	for _, r := range model.PlanRows(bindingsOf(t, mixed(t))) {
		testkit.NotEqual(t, r.ID.Qualifier, "",
			"a family-scoped identity carries the interface's word, or two "+
				"interfaces in one package mint one identity")
	}
}

// The call and the declaration are one contribution in two regions, so
// they agree about the identifier, the parameter and what comes back.
//
// An expression naming a function nobody declared is a compile error
// over generated code a consumer cannot edit — the failure this pairing
// makes unrepresentable.
func TestTheCallAndTheDeclarationAgree(t *testing.T) {
	t.Parallel()

	b := bindingsOf(t, mixed(t))
	testkit.NotEqual(t, b.RowsFuncName(), "",
		"the function the contributed expression names")
	testkit.NotContains(t, b.RowsFuncName(), "ModelChecks",
		"and not the index group type's name, which a call to it would read as a conversion")
}
