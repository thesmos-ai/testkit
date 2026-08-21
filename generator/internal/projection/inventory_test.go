// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// scanAppendSmoke is the design document's populated sample, as data.
func scanAppendSmoke() projection.CheckPlan {
	return projection.CheckPlan{
		ID:    projection.IDPlan{Method: "Append", Seg: suite.SegSmoke},
		Class: suite.ClassSmoke,
		Claim: "Append survives a call with a derived entry",
		Body: projection.SmokeSurvives{Call: projection.CallPlan{
			Method: "Append",
			Args:   []projection.Expr{projection.ExprCtx, projection.FixtureCall(projection.ExprFixture, "entry")},
		}},
		Falsifiable: suite.Proven(),
		Defect:      projection.StubPanic{Option: projection.OptionName("Log", "Append")},
	}
}

// parityCase mutates the valid sample into one refusal shape.
type parityCase struct {
	name   string
	mutate func(*projection.CheckPlan)
}

func (c parityCase) Name() string { return c.name }

//nolint:thelper // the case body is the test, not a helper; see core/lawid
func TestInventoryVerifyHoldsTheParityRules(t *testing.T) {
	t.Parallel()

	t.Run("the design sample verifies", func(t *testing.T) {
		t.Parallel()
		inv := projection.Inventory{Iface: "Log", Token: "log", Checks: []projection.CheckPlan{scanAppendSmoke()}}
		testkit.NoError(t, inv.Verify(), "the populated sample from the design document is valid")
	})

	testkit.TableTest(t, []parityCase{
		{"proven without a defect", func(c *projection.CheckPlan) { c.Defect = nil }},
		{"argued yet with a defect", func(c *projection.CheckPlan) { c.Falsifiable = suite.Argued("x") }},
		{"argued with no argument", func(c *projection.CheckPlan) {
			c.Defect = nil
			c.Falsifiable = suite.Falsifiability{State: suite.FalsifiableArgued}
		}},
		{"no claim", func(c *projection.CheckPlan) { c.Claim = "" }},
		{"no body", func(c *projection.CheckPlan) { c.Body = nil }},
		{"underived falsifiability", func(c *projection.CheckPlan) {
			c.Falsifiable = suite.Falsifiability{}
			c.Defect = nil
		}},
	}, func(t *testing.T, tc parityCase) {
		c := scanAppendSmoke()
		tc.mutate(&c)
		inv := projection.Inventory{Iface: "Log", Token: "log", Checks: []projection.CheckPlan{c}}
		testkit.Error(t, inv.Verify(), "the parity rules refuse at generation time")
	})

	t.Run("a check derived twice is refused", func(t *testing.T) {
		t.Parallel()
		inv := projection.Inventory{
			Iface: "Log", Token: "log",
			Checks: []projection.CheckPlan{scanAppendSmoke(), scanAppendSmoke()},
		}
		err := inv.Verify()
		testkit.Error(t, err, "duplicate derivation must be refused")
		testkit.Contains(t, err.Error(), "twice", "the refusal names the duplication")
	})
}
