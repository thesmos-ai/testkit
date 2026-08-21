// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// The capability-door rules — A10's first half: a check declares what
// it needs, and the harness carries only what some check declared.
// The row constructors consult this table; the harness projection
// aggregates the declared doors into the generated type's fields.
//
// The recover door joins with the sim family's licensing directive —
// no rule may demand a capability no row can earn.

// capRule answers one class's capability doors from the stamps that
// licensed the row.
type capRule func(f Iface) []projection.NeedPlan

// capRules is the door table, keyed by the class a row reports under.
// A class without a row demands nothing, which is A10's default: the
// harness surface never grows on speculation.
func capRules() map[vocab.Class]capRule {
	return map[vocab.Class]capRule{
		vocab.ClassClocked: clockDoor,
		vocab.ClassPoison:  induceDoor,
	}
}

// capsFor resolves one row's capability doors, nil for the classes
// that walk in unaided.
func capsFor(f Iface, class vocab.Class) []projection.NeedPlan {
	rule, gated := capRules()[class]
	if !gated {
		return nil
	}
	return rule(f)
}

// clockDoor: a clocked row moves time, so the subject must be
// constructible on a controlled clock.
//
// Derived here while the model tier's emit layer is being replaced. The
// field is that tier's to contribute — see the harness's fields region —
// and this goes when it does.
func clockDoor(Iface) []projection.NeedPlan {
	return []projection.NeedPlan{{Capability: vocab.CapClock}}
}

// induceDoor: a poison row needs the subject put into its sentinel
// state on demand — the door's value is the stamped sentinel, the
// same declaration that licensed the law and planted its defect.
func induceDoor(f Iface) []projection.NeedPlan {
	for _, m := range f.Methods {
		if v, ok := m.MixinParam(MixinAfterClose, MixinAfterCloseSentinel); ok && v != "" {
			return []projection.NeedPlan{{Capability: vocab.CapInduce, Value: projection.Expr(v)}}
		}
	}
	return nil
}
