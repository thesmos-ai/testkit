// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// ModelRowKinds is the set of check-body shapes this tier renders.
//
// The harness generator plans them and lists them under Withheld,
// because a row planned by one tier and rendered by another is exactly
// what Withheld means. Planning cannot be split by tier: the harness's
// own fields are derived from the plans — a clocked law's row is what
// puts the Clock on the harness — so the row and the field it needs have
// to be worked out together. See [projection.HarnessOf].
func ModelRowKinds() []projection.BodyKind {
	return []projection.BodyKind{
		projection.KindLawLeg,
		projection.KindDifferentialLeg,
		projection.KindSimLeg,
	}
}

// ModelRows selects the rows this tier renders out of the whole planned
// inventory, in plan order.
//
// Order is the inventory's, which is the derivers' declaration order, so
// the rendered rows and the header that describes them agree without
// either sorting.
func ModelRows(inv projection.Inventory) []projection.CheckPlan {
	kinds := ModelRowKinds()
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
