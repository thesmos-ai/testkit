// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

import (
	"fmt"

	"go.thesmos.sh/testkit/engine/suite"
)

// Inventory is one interface's complete check set plus the identity
// the artifacts share. Every emitted artifact is a projection of this
// value; nothing renders from anywhere else.
type Inventory struct {
	// Iface is the interface's exported name ("Log"); Token is the Go
	// identifier qualifier every emitted declaration is named from
	// ("log", "paginatedReader").
	//
	// Not the ID qualifier, which is a slug and diverges the moment an
	// interface name has two words — a family-scoped identity composes
	// from [Iface.Qualifier], and this field named that once, which is
	// how "model/paginatedReader/laws" reached the grammar.
	Iface string
	Token string

	Checks []CheckPlan
}

// Verify holds the inventory to its own invariants — the parity and
// shape rules every downstream projection assumes. A deriver bug
// surfaces here, at generation time, with the check named; silence
// past this point would become a silent-green in a consumer's run.
func (inv *Inventory) Verify() error {
	seen := map[suite.ID]bool{}
	for _, c := range inv.Checks {
		id, err := c.ID.Render()
		if err != nil {
			return err
		}
		if seen[id] {
			return fmt.Errorf("projection: %s derived twice", id)
		}
		seen[id] = true
		if err := suite.ValidateID(id); err != nil {
			return fmt.Errorf("projection: %s: %w", id, err)
		}
		if c.Claim == "" {
			return fmt.Errorf("projection: %s has no claim", id)
		}
		if c.Body == nil {
			return fmt.Errorf("projection: %s has no body", id)
		}
		switch c.Falsifiable.State {
		case suite.FalsifiableProven:
			if c.Defect == nil {
				return fmt.Errorf("projection: %s claims Proven with no planted defect", id)
			}
		case suite.FalsifiableArgued:
			if c.Defect != nil {
				return fmt.Errorf(
					"projection: %s is Argued yet plants a defect — the evidence exists, so the claim is owed",
					id,
				)
			}
			if c.Falsifiable.Why == "" {
				return fmt.Errorf("projection: %s is Argued with no argument", id)
			}
		default:
			return fmt.Errorf("projection: %s is neither Proven nor Argued; an underived state cannot be emitted", id)
		}
	}
	return nil
}
