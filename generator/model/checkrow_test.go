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

// A row carries everything the harness generator's own rows carry.
//
// They land in one []suite.Check and a consumer reading the report
// cannot tell which generator wrote which, so a row that carried less
// would be a row the runner treats differently.
func TestCheckRowsCarryTheWholePlan(t *testing.T) {
	t.Parallel()

	rows := model.CheckRows("mixed", []projection.CheckPlan{{
		ID:          projection.IDPlan{Family: vocab.FamilyModel, Seg: "ttl-expiry"},
		Class:       vocab.ClassClocked,
		Claim:       "an entry stops being readable once its lifetime has run out",
		Falsifiable: vocab.Proven(),
		Needs:       []projection.NeedPlan{{Capability: vocab.CapClock}},
	}})

	testkit.Len(t, rows, 1, "one row per plan")
	r := rows[0]
	testkit.Equal(t, r.Accessor, "TTLExpiry", "the index method, family fixed and segment varying")
	testkit.Equal(t, r.AssertName, "mixedAssertTTLExpiry", "the token and the segment")
	testkit.Equal(t, r.Claim, "an entry stops being readable once its lifetime has run out",
		"carried verbatim — the plan's claim is what the census measured")
	testkit.True(t, r.Proven, "a plan with a defect behind it may be stamped Proven")
	testkit.Len(t, r.Needs, 1, "and the capability it demands of the harness")
	testkit.Equal(t, r.Needs[0].Const, "CapClock",
		"a door the runtime names is spelled through its constant")
}

// Every door a plan carries reaches the row, not just the first.
//
// A law can read a clock and a consumer-supplied fact at once, and a row
// naming only one of them is a row the runner admits against a subject
// that cannot serve it — the check then fails inside the body, naming a
// nil closure instead of the field that would have armed it.
func TestCheckRowsCarryEveryDoor(t *testing.T) {
	t.Parallel()

	rows := model.CheckRows("mixed", []projection.CheckPlan{{
		ID:    projection.IDPlan{Family: vocab.FamilyModel, Seg: "windowed"},
		Class: vocab.ClassLaws,
		Needs: []projection.NeedPlan{
			{Capability: vocab.CapClock},
			{Capability: "history"},
		},
	}})

	testkit.Len(t, rows[0].Needs, 2, "both doors, in plan order")
	testkit.Equal(t, rows[0].Needs[1].Const, "",
		"a door the runtime declares no constant for has none to spell")
	testkit.Equal(t, rows[0].Needs[1].Name, "history",
		"so the row names it as the literal the harness answers under")
}

// An argued row is not stamped proven.
func TestCheckRowsKeepTheFalsifiabilityVerdict(t *testing.T) {
	t.Parallel()

	rows := model.CheckRows("mixed", []projection.CheckPlan{{
		ID:          projection.IDPlan{Family: vocab.FamilyModel, Seg: "laws"},
		Falsifiable: vocab.Argued("no defect template plants this yet"),
	}})

	testkit.False(t, rows[0].Proven,
		"claiming proof without the evidence is refused, in both directions")
}

// Nothing is dropped, because the index names every row this tier owns.
func TestCheckRowsDropNothing(t *testing.T) {
	t.Parallel()

	plans := []projection.CheckPlan{
		{ID: projection.IDPlan{Family: vocab.FamilyModel, Seg: "laws"}},
		{ID: projection.IDPlan{Family: vocab.FamilyModel, Seg: "agrees"}},
		{ID: projection.IDPlan{Family: vocab.FamilySim, Seg: "recovery"}},
	}

	rows := model.CheckRows("mixed", plans)
	testkit.Len(t, rows, len(plans),
		"a plan dropped here leaves the index naming a check nothing emits")
	testkit.Equal(t, rows[2].AssertName, "mixedAssertRecovery", "the sim family is spelled the same way")

	testkit.Len(t, model.CheckRows("mixed", nil), 0, "and no plans is no rows")
}
