// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// harnessCase is one check set and the harness surface it demands.
type harnessCase struct {
	name   string
	checks []projection.CheckPlan
	want   projection.HarnessPlan
}

func (c harnessCase) Name() string { return c.name }

//nolint:thelper // the case body is the test, not a helper; see core/lawid
func TestHarnessCarriesOnlyWhatChecksDemand(t *testing.T) {
	t.Parallel()

	clocked := projection.CheckPlan{
		ID:    projection.IDPlan{Family: suite.FamilyModel, Qualifier: "store", Seg: suite.SegLaws},
		Needs: []projection.NeedPlan{{Capability: suite.CapClock}},
	}
	induced := projection.CheckPlan{
		ID:    projection.IDPlan{Family: suite.FamilyModel, Qualifier: "store", Seg: suite.SegPoison},
		Needs: []projection.NeedPlan{{Capability: suite.CapInduce, Value: projection.Expr("kv.ErrClosed")}},
	}
	seededHit := projection.CheckPlan{ID: projection.IDPlan{Method: "Lookup", Seg: suite.SegHit}}
	plainSmoke := projection.CheckPlan{ID: projection.IDPlan{Method: "Get", Seg: suite.SegSmoke}}

	testkit.TableTest(t, []harnessCase{
		{
			"a bare check set demands a bare harness",
			[]projection.CheckPlan{plainSmoke},
			projection.HarnessPlan{Iface: "Store"},
		},
		{
			"a clocked check opens the clock field",
			[]projection.CheckPlan{plainSmoke, clocked},
			projection.HarnessPlan{Iface: "Store", Clock: true},
		},
		{
			"an induced check opens the induction map",
			[]projection.CheckPlan{induced},
			projection.HarnessPlan{Iface: "Store", Induce: true},
		},
		{
			"a seeded claim opens the seed seam",
			[]projection.CheckPlan{seededHit},
			projection.HarnessPlan{Iface: "Store", Seeded: true},
		},
		{
			"doors accumulate without duplicating",
			[]projection.CheckPlan{clocked, clocked, induced, seededHit},
			projection.HarnessPlan{Iface: "Store", Clock: true, Induce: true, Seeded: true},
		},
	}, func(t *testing.T, tc harnessCase) {
		testkit.Equal(t, projection.HarnessOf("Store", tc.checks, nil), tc.want,
			"the harness surface is exactly the declared doors — A10, structurally")
	})
}
