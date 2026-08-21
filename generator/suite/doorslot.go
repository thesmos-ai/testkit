// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/naming"
)

// The harness-fields region: a capability field another tier contributes.
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
// HarnessFields returns the region, creating it on first reach.
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
func (c *Contract) Slot(name string) *sdk.Slot {
	if name == naming.SlotHarnessFields {
		return c.HarnessFields()
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
