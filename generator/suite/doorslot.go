// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/naming"
)

// HarnessFields returns the region another tier contributes capability
// fields into, creating it on first reach.
//
// A door is a field on the generated harness: the clock a check moves
// instead of waiting on, the induction that puts a subject into a
// failure state on demand. This generator emits the harness and knows
// nothing about which doors exist — every capability in the table today
// belongs to a check this tier does not write, and the sentence
// explaining why a consumer must fill one is a fact about that tier's
// checks, not about the harness.
//
// So the harness carries a region and renders whatever is in it. A tier
// that needs a door contributes the field and its own prose; a package
// with no such tier renders an empty region and imports nothing extra.
//
// No element-kind constraint: a contributor brings its own emit kind and
// its own template, which is the arrangement [sdk.NewSlot] documents for
// heterogeneous content. What the slot guarantees either way is append
// order and attribution — a field's provenance names the plugin that put
// it there, so a reader of the generated harness can tell which tier
// asked for what.
func (c *Contract) HarnessFields() *sdk.Slot {
	if c.doors == nil {
		c.doors = sdk.NewSlot(naming.SlotHarnessFields, "")
		c.doors.Owner = c
	}
	return c.doors
}

// Slot satisfies [sdk.SlotHost] so the backend's `slot` template helper
// reaches the region by name.
// A region reached by a name this switch does not know gets a fresh
// empty one, which renders nothing at all — so a name added to the
// vocabulary and not added here fails by omission rather than by error.
// TestEveryRegionIsReachable holds the two together.
func (c *Contract) Slot(name string) *sdk.Slot {
	switch name {
	case naming.SlotHarnessFields:
		return c.HarnessFields()
	case naming.SlotHarnessLowering:
		return c.HarnessLowering()
	case naming.SlotSuiteChecks:
		return c.Rows()
	case naming.SlotSuiteDecls:
		return c.Decls()
	}
	return sdk.NewSlot(name, "")
}

var _ sdk.SlotHost = (*Contract)(nil)

// FieldsSlotName is the region's name, for the template's `slot` helper.
//
// Read off the value rather than spelled in the template, because a
// template literal is a second home for the name and the misspelling it
// invites fails by rendering nothing at all: the helper mints an empty
// region under the near-miss name and the harness comes out without the
// fields, which compiles.
func (*Contract) FieldsSlotName() string { return naming.SlotHarnessFields }

// HarnessLowering returns the region a contributed field's assignment
// onto the runtime subject lands in.
//
// Paired with the fields region on purpose. A field with no lowering is a
// place for a consumer to write a value nothing reads, which is what the
// harness carried for a clock until this pair existed.
func (c *Contract) HarnessLowering() *sdk.Slot {
	if c.lowering == nil {
		c.lowering = sdk.NewSlot(naming.SlotHarnessLowering, "")
		c.lowering.Owner = c
	}
	return c.lowering
}

// LoweringSlotName is the region's name, for the template's helper.
func (*Contract) LoweringSlotName() string { return naming.SlotHarnessLowering }

// ClockPkg surfaces the clock package to the templates, whose harness
// spells its test clock in type position.
func (*Contract) ClockPkg() string { return ClockPkg }
