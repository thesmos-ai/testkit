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

// The rows this tier renders are selected out of the whole planned
// inventory, and nothing else comes with them.
//
// The harness generator plans every row because its own fields are
// derived from the plans — a clocked law's row is what puts Clock on the
// harness. What splits by tier is who renders, so this is a filter over
// one derivation rather than a second derivation.
func TestModelRowsSelectsTheLegKinds(t *testing.T) {
	t.Parallel()

	inv := projection.Inventory{Checks: []projection.CheckPlan{
		{ID: projection.IDPlan{Method: "Get", Seg: vocab.SegSmoke}, Body: projection.SmokeSurvives{}},
		{ID: projection.IDPlan{Seg: "laws"}, Body: projection.LawLeg{}},
		{ID: projection.IDPlan{Method: "Get", Seg: vocab.SegMiss}, Body: projection.ZeroOnMiss{}},
		{ID: projection.IDPlan{Seg: "agrees"}, Body: projection.DifferentialLeg{}},
		{ID: projection.IDPlan{Seg: "recovery"}, Body: projection.SimLeg{}},
	}}

	got := model.ModelRows(inv)

	testkit.Len(t, got, 3, "the three leg kinds, and none of the harness generator's own")
	testkit.Equal(t, got[0].ID.Seg, "laws", "in the inventory's order")
	testkit.Equal(t, got[1].ID.Seg, "agrees", "which is the derivers' declaration order")
	testkit.Equal(t, got[2].ID.Seg, "recovery", "so the rows and the header agree without sorting")
}

// A plan with no body is skipped rather than dereferenced.
//
// The state is real: a deriver that refused a row still files the
// refusal, and a plan can reach the inventory carrying an ID and no body.
func TestModelRowsSkipsABodylessPlan(t *testing.T) {
	t.Parallel()

	inv := projection.Inventory{Checks: []projection.CheckPlan{
		{ID: projection.IDPlan{Seg: "laws"}},
		{ID: projection.IDPlan{Seg: "agrees"}, Body: projection.DifferentialLeg{}},
	}}

	got := model.ModelRows(inv)
	testkit.Len(t, got, 1, "only the one with a body to render")
}

// An inventory with nothing this tier owns yields nothing, not an empty
// slice somebody has to guard.
func TestModelRowsOnASuiteOnlyInterface(t *testing.T) {
	t.Parallel()

	inv := projection.Inventory{Checks: []projection.CheckPlan{
		{ID: projection.IDPlan{Method: "Get", Seg: vocab.SegSmoke}, Body: projection.SmokeSurvives{}},
	}}

	testkit.Len(t, model.ModelRows(inv), 0,
		"an interface carrying no model directive plans no model row")
	testkit.Len(t, model.ModelRows(projection.Inventory{}), 0,
		"and an empty inventory is the same answer")
}

// The kind set is closed and every member is a body variant the
// projection declares, so a kind renamed there fails here rather than
// silently stopping the selection.
func TestModelRowKindsAreDeclaredBodyKinds(t *testing.T) {
	t.Parallel()

	declared := projection.BodyKinds()
	for _, k := range model.ModelRowKinds() {
		testkit.Contains(t, declared, k,
			"this tier claims to render "+string(k)+", which projection does not declare")
	}
}
