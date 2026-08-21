// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

import "go.thesmos.sh/testkit/engine/suite"

// HarnessPlan is the generated harness type's surface for one
// interface: the capability fields the emitted check set demands, and
// the seed seam where the claims read a received corpus. A projection
// of the inventory — A10's rule made structural: no OnClock without a
// clocked check, no Induce without an induced sentinel, no Recover
// without a sim row, no seed constructor without a seeded claim. The
// harness surface cannot grow on speculation because there is nothing
// here to set except from a check.
type HarnessPlan struct {
	// Iface is the subject interface's exported name; the emitted
	// identifiers derive through the naming policy, never here.
	Iface string

	// Clock, Induce and Recover mirror the capability doors some
	// check declared through its Needs.
	Clock   bool
	Induce  bool
	Recover bool

	// Seeded marks the seed-seam constructor pair: the harness
	// receives the corpus the pools derive, because a reader-only
	// subject cannot be populated through the surface under test.
	Seeded bool
}

// HarnessOf projects the harness surface from the interface's check
// plans. Reading the plans rather than the stamps is the point: a
// capability nothing checks is a field nobody may demand, and the
// derivation cannot disagree with the run that gates on it.
func HarnessOf(iface string, checks []CheckPlan) HarnessPlan {
	p := HarnessPlan{Iface: iface}
	for _, c := range checks {
		for _, n := range c.Needs {
			switch n.Capability {
			case suite.CapClock:
				p.Clock = true
			case suite.CapInduce:
				p.Induce = true
			case suite.CapRecover:
				p.Recover = true
			}
		}
		if c.ID.Seg == suite.SegHit || c.ID.Seg == suite.SegCount {
			p.Seeded = true
		}
	}
	return p
}
