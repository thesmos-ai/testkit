// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

import "go.thesmos.sh/testkit/engine/suite"

// HarnessPlan is the generated harness type's surface for one
// interface: the seed seam where the claims read a received corpus.
//
// It carries no capability. Every capability in testkit's table — a
// clock a check advances, an induction that provokes a failure state, a
// recovery across a crash — is demanded by a check the harness generator
// does not write, and a field exists because some check needs it. So the
// tier whose check needs one contributes the field, through the
// harness's fields region, and this projects only what this tier's own
// checks imply.
type HarnessPlan struct {
	// Iface is the subject interface's exported name; the emitted
	// identifiers derive through the naming policy, never here.
	Iface string

	// Seeded marks the seed-seam constructor pair: the harness
	// receives the corpus the pools derive, because a reader-only
	// subject cannot be populated through the surface under test.
	Seeded bool
}

// HarnessOf projects the harness surface from the interface's check
// plans. Reading the plans rather than the stamps is the point: the
// surface cannot grow on speculation because there is nothing here to
// set except from a check.
func HarnessOf(iface string, checks []CheckPlan) HarnessPlan {
	p := HarnessPlan{Iface: iface}
	for _, c := range checks {
		if c.ID.Seg == suite.SegHit || c.ID.Seg == suite.SegCount {
			p.Seeded = true
		}
	}
	return p
}
