// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

func TestIDPlanRenders(t *testing.T) {
	t.Parallel()

	t.Run("method scope through the vocabulary", func(t *testing.T) {
		t.Parallel()
		got, err := projection.IDPlan{Method: "Append", Seg: suite.SegSmoke}.Render()
		testkit.NoError(t, err, "a method-scoped plan is well formed")
		testkit.Equal(t, got, suite.ID("Append/smoke"), "method IDs render Method/seg")
	})

	t.Run("family scope carries its qualifier", func(t *testing.T) {
		t.Parallel()
		got, err := projection.IDPlan{Family: suite.FamilyModel, Qualifier: "log", Seg: suite.SegLaws}.Render()
		testkit.NoError(t, err, "a qualified family plan is well formed")
		testkit.Equal(t, got, suite.ID("model/log/laws"), "family IDs qualify unconditionally")
	})
}

func TestIDPlanRefusesMalformedPlans(t *testing.T) {
	t.Parallel()

	t.Run("both scopes", func(t *testing.T) {
		t.Parallel()
		_, err := projection.IDPlan{Method: "Append", Family: suite.FamilyModel, Qualifier: "log", Seg: "x"}.Render()
		testkit.Error(t, err, "a plan naming both scopes is a deriver bug")
	})

	t.Run("unqualified family", func(t *testing.T) {
		t.Parallel()
		_, err := projection.IDPlan{Family: suite.FamilyModel, Seg: suite.SegLaws}.Render()
		testkit.Error(t, err,
			"qualification is unconditional; an unqualified family plan is a deriver bug")
	})

	t.Run("empty plan", func(t *testing.T) {
		t.Parallel()
		_, err := projection.IDPlan{}.Render()
		testkit.Error(t, err, "an empty plan renders nothing")
	})
}
