// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// doorsFor works out which capabilities this interface's declarations
// imply, whoever ends up checking them.
//
// A capability door is a field on the harness: the clock a check moves
// instead of waiting, the induction that puts the subject into a failure
// state. The harness is this generator's output, so the field is this
// generator's to emit — but WHICH doors are needed is a fact about the
// classifications, and [tiers] is where that fact lives. Both the rule
// selection and the leg it reports under come from there.
//
// Derived from the stamps rather than from a check plan, which is the
// change that lets this generator plan only its own rows. A door read
// off a row is a door that exists only while the row does, and the rows
// that need these are not this tier's.
//
// The narrower reading [projection.HarnessOf] makes over the plans still
// stands beside it: a capability some check declares is one the harness
// must carry. This adds the ones no check of THIS tier declares and the
// subject nonetheless needs, which a consumer would otherwise have no
// field to supply.
func doorsFor(f Iface) []projection.NeedPlan {
	seen := map[string]bool{}
	var out []projection.NeedPlan
	// Through [selectLaws] rather than [tiers.Select] directly: the
	// selection is the catalogue PLUS the extra rules, and one of those
	// is what licenses a poison law from a declared sentinel. A second
	// walk over the catalogue alone would open the clock's door and not
	// the induction's, which is a harness that satisfies some of what a
	// subject is about to be asked for.
	for _, s := range selectLaws(f) {
		class, own := tiers.LegOf(s.Law)
		if !own {
			// A bundled law rides the shared sequences under one claim
			// and asks for nothing of its own.
			continue
		}
		if _, worded := lawid.ClaimOf(s.Law); !worded {
			// No claim, so no row will state it, so nothing will use the
			// door. [projection.HarnessOf]'s rule holds here too: a
			// capability nothing checks is a field nobody may demand.
			continue
		}
		for _, n := range capsFor(f, class) {
			if seen[string(n.Capability)] {
				continue
			}
			seen[string(n.Capability)] = true
			out = append(out, n)
		}
	}
	return out
}
