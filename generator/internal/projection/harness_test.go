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

	seededHit := projection.CheckPlan{ID: projection.IDPlan{Method: "Lookup", Seg: suite.SegHit}}
	plainSmoke := projection.CheckPlan{ID: projection.IDPlan{Method: "Get", Seg: suite.SegSmoke}}

	testkit.TableTest(t, []harnessCase{
		{
			"a bare check set demands a bare harness",
			[]projection.CheckPlan{plainSmoke},
			projection.HarnessPlan{Iface: "Store"},
		},
		{
			"a seeded claim opens the seed seam",
			[]projection.CheckPlan{seededHit},
			projection.HarnessPlan{Iface: "Store", Seeded: true},
		},
		{
			// A capability declared by a check of another tier reaches
			// the harness through its fields region, not through here.
			"a capability a check declares opens no field",
			[]projection.CheckPlan{{
				ID:    projection.IDPlan{Family: suite.FamilyModel, Qualifier: "store", Seg: suite.SegLaws},
				Needs: []projection.NeedPlan{{Capability: suite.CapClock}},
			}},
			projection.HarnessPlan{Iface: "Store"},
		},
	}, func(t *testing.T, tc harnessCase) {
		testkit.Equal(t, projection.HarnessOf("Store", tc.checks), tc.want,
			"the harness surface is exactly what this tier's own claims imply")
	})
}
