// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/projection"
)

// LegTemplates maps each check-body kind this tier renders to the emit
// kinds its dispatch can produce for it.
//
// An emit kind IS its template's name, so this table and the embedded
// template set can be held equal — which is the check the composition
// root cannot make. That census reads [RowKinds] from both tiers and
// catches a kind NEITHER claims; a kind this tier claims and spells no
// template for looks identical to one it renders, because a claim is all
// the root can see. Deriving the claim from the dispatch is what closes
// that: the list below is now a fact about the templates rather than a
// sentence written beside them.
func LegTemplates() map[projection.BodyKind][]sdk.Kind {
	return map[projection.BodyKind][]sdk.Kind{
		projection.KindLawLeg:          {KindLawsLeg, KindOwnLeg, KindClockedLeg},
		projection.KindDifferentialLeg: {KindDifferentialLeg},
		projection.KindConcurrentLeg:   {KindConcurrent},
		projection.KindSimLeg:          {KindSimLeg},
	}
}

// RowKinds is the set of check-body shapes this tier renders, in the
// projection's declaration order.
//
// The harness generator plans them and lists them under Withheld,
// because a row planned by one tier and rendered by another is exactly
// what Withheld means. Planning cannot be split by tier: the harness's
// own fields are derived from the plans — a clocked law's row is what
// puts the Clock on the harness — so the row and the field it needs have
// to be worked out together. See [projection.HarnessOf].
func RowKinds() []projection.BodyKind {
	rendered := LegTemplates()
	out := make([]projection.BodyKind, 0, len(rendered))
	for _, k := range projection.BodyKinds() {
		if _, ours := rendered[k]; ours {
			out = append(out, k)
		}
	}
	return out
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
