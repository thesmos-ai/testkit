// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// RowKinds is the set of check-body shapes this tier renders.
//
// The harness generator plans them and lists them under Withheld,
// because a row planned by one tier and rendered by another is exactly
// what Withheld means. Planning cannot be split by tier: the harness's
// own fields are derived from the plans — a clocked law's row is what
// puts the Clock on the harness — so the row and the field it needs have
// to be worked out together. See [projection.HarnessOf].
func RowKinds() []projection.BodyKind {
	return []projection.BodyKind{
		projection.KindLawLeg,
		projection.KindDifferentialLeg,
		projection.KindSimLeg,
	}
}

// Rows selects the rows this tier renders out of a planned inventory, in
// plan order.
//
// Kept for a reader of somebody else's inventory. This tier plans its own
// rows now — see planRows — because the reason they were planned
// elsewhere is gone: the harness's capability fields were projected from
// the plans, so a clocked law's row had to be worked out with the field
// it opened, and this tier contributes that field itself.
func Rows(inv projection.Inventory) []projection.CheckPlan {
	kinds := RowKinds()
	var out []projection.CheckPlan
	for _, c := range inv.Checks {
		if c.Body == nil {
			continue
		}
		for _, k := range kinds {
			if c.Body.BodyKind() == k {
				out = append(out, c)
				break
			}
		}
	}
	return out
}
