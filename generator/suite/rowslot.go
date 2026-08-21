// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/naming"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// Rows returns the region another tier contributes its checks into,
// creating it on first reach.
//
// A contribution renders as one expression yielding a slice of checks,
// which the run surface appends. That is the whole composition: the
// package has one Suite, and a claim's rows sit beside the rows they are
// compared against rather than in a second file a reader has to know to
// open.
func (c *Contract) Rows() *sdk.Slot {
	if c.rows == nil {
		c.rows = sdk.NewSlot(naming.SlotSuiteChecks, "")
		c.rows.Owner = c
	}
	return c.rows
}

// RowsSlotName is the region's name, for the template's helper.
func (*Contract) RowsSlotName() string { return naming.SlotSuiteChecks }

// Decls returns the file's package-level region, creating it on first
// reach — where a contributed expression's function is declared.
func (c *Contract) Decls() *sdk.Slot {
	if c.decls == nil {
		c.decls = sdk.NewSlot(naming.SlotSuiteDecls, "")
		c.decls.Owner = c
	}
	return c.decls
}

// DeclsSlotName is the region's name, for the template's helper.
func (*Contract) DeclsSlotName() string { return naming.SlotSuiteDecls }

// Rowed is what a contribution into [Contract.Rows] must answer.
//
// The rendered expression is not enough on its own. Every check in the
// package needs an entry in the typed index — a consumer drops a check
// by naming it, and a check the index cannot name is one they cannot
// drop, prove, or run alone. The index is this generator's to emit, so a
// contributing tier hands over the plans and this generator names them.
//
// An interface rather than a type, so a contribution stays the
// contributor's own emit value with its own template. This generator
// asks it one question and renders it; it does not know what it is.
type Rowed interface {
	// CheckPlans are the checks the contribution's expression yields, in
	// the order it yields them.
	CheckPlans() []projection.CheckPlan
}

// CheckIndex is the typed index over every check the package emits —
// this generator's rows and every contributed one.
//
// A method rather than the [Contract.Index] field, because a
// contribution arrives after this value is queued: a contributing tier
// reads the queued value, so anything it adds is only visible once every
// generator has run, which is render time. The field stays for what this
// generator derived alone.
//
// A malformed contributed ID falls back to this generator's own index
// rather than failing the render. The run's own invariants already hold
// the emitted set to the manifest, and the lock verifier reports an index
// that names nothing far better than a template error names it.
func (c *Contract) CheckIndex() projection.IndexPlan {
	contributed := c.contributedPlans()
	if len(contributed) == 0 {
		return c.Index
	}
	merged := make([]projection.CheckPlan, 0, len(c.Inventory.Checks)+len(contributed))
	merged = append(merged, emittedPlans(c.Checks)...)
	merged = append(merged, contributed...)

	index, err := projection.IndexOf(projection.Inventory{Checks: merged})
	if err != nil {
		return c.Index
	}
	return index
}

// ContributedIDs are the identities the contributed rows report under,
// for the manifest and the lock.
func (c *Contract) ContributedIDs() ([]string, error) {
	return emittedIDs(projection.Inventory{Checks: c.contributedPlans()})
}

// contributedPlans is every plan the regions carry, in contribution
// order.
//
// Order is the slot's, which is registration order across plugins and
// FIFO within one — so the index and the appended expressions agree
// without either sorting.
func (c *Contract) contributedPlans() []projection.CheckPlan {
	if c.rows == nil {
		return nil
	}
	var out []projection.CheckPlan
	for i := range c.rows.Len() {
		if r, rowed := c.rows.At(i).(Rowed); rowed {
			out = append(out, r.CheckPlans()...)
		}
	}
	return out
}
