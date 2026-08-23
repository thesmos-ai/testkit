// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
)

// The row-binding vocabulary. A generated package's check table lowers
// every row onto [Check] with the same four judgements — which body,
// which ID, which method, what falsifiability — and before this file each
// interface's bind restated the judgements and their failure prose. The
// enumeration (which body fields exist, what each receives) is
// irreducibly per-interface and stays in the template; the rules and the
// wording live here once.

// NameSet is an interface's method set, held for validating a row's
// Method field: a misspelled method starts uppercase too, so the ID
// grammar alone would file a check under a method that does not exist.
// The owner's name and the sorted listing feed the failure message, so
// the prose cannot drift from the set it describes.
type NameSet struct {
	owner string
	names []string
	has   map[string]bool
}

// NewNameSet builds the set for one owner — "Store" — and its methods.
func NewNameSet(owner string, names ...string) NameSet {
	has := make(map[string]bool, len(names))
	for _, n := range names {
		has[n] = true
	}
	return NameSet{owner: owner, names: slices.Sorted(maps.Keys(has)), has: has}
}

// Owner returns the name the set was built for.
func (s NameSet) Owner() string { return s.owner }

// Has reports whether the set holds name.
func (s NameSet) Has(name string) bool { return s.has[name] }

// List renders the set for a failure message, sorted.
func (s NameSet) List() string { return strings.Join(s.names, ", ") }

// RowID composes a method-scoped row ID, holding the row to the two rules
// its body shape imposes: the field needs a Method, and the Method must
// name a method the interface has. field is the body field's name — Run,
// Prop — because the fix belongs to the field the consumer set.
func RowID(field, method, name string, methods NameSet) (ID, error) {
	if method == "" {
		return "", fmt.Errorf(
			"check %q sets %s and no Method; %s's ID needs the method scope", name, field, field,
		)
	}
	if !methods.Has(method) {
		return "", fmt.Errorf(
			"check %q names method %q, which %s does not have (methods: %s)",
			name, method, methods.Owner(), methods.List(),
		)
	}
	return MethodID(method, name), nil
}

// HandRowID composes the ID for a row whose body fixes its own scope — a
// RunWith claim with no method of its own.
func HandRowID(name string) ID { return FamilyID(FamilyHand, "", name) }

// OneBody enforces the row contract's core rule: exactly one body field.
// fields is the interface's own listing, because which Prop* sugar exists
// is per-interface and the message must offer what the row can actually
// set.
func OneBody(row string, count int, fields string) error {
	if count == 1 {
		return nil
	}
	return fmt.Errorf("check %q sets %d bodies; set exactly one of %s", row, count, fields)
}

// ProvenCheck builds a deterministic generated check with the Proven
// stamp — the shape every signature-tier emission takes. The stamp is
// part of the constructor deliberately: the proofs file plants a defect
// for each of these, and the parity gate refuses the stamp without one,
// so a constructor that let the stamp drift from the shape would let a
// check ship unproven while reading as proven.
func ProvenCheck[S any](id ID, class Class, claim string, run func(tb testing.TB, s S)) Check[S] {
	return Check[S]{ID: id, Class: class, Claim: claim, Falsifiable: Proven(), Run: run}
}

// ArguedCheck builds the same shape with the Argued stamp and the
// argument for it.
//
// The companion to [ProvenCheck], and the reason both constructors carry
// the stamp rather than taking it: the parity gate refuses a defect for a
// check that does not claim Proven as firmly as it refuses the claim
// without one. A generator that spells a check's body and not yet its
// defect has to say so here, or the proofs file it writes beside this one
// fails on the row it left out — which is the gate working, reported
// against the wrong file.
func ArguedCheck[S any](
	id ID, class Class, claim, why string, run func(tb testing.TB, s S),
) Check[S] {
	return Check[S]{ID: id, Class: class, Claim: claim, Falsifiable: Argued(why), Run: run}
}

// At records how far this check looks before it passes, and returns the
// check so a constructor call can carry it.
//
// A method rather than a parameter on [ProvenCheck] and [ArguedCheck].
// Those two are called from generated files that already exist, and a
// new parameter would break every one of them at once — including files
// this repository does not own. What it costs is that a caller CAN omit
// it; what limits that cost is the zero value, which is the weakest of
// the three, so a check that never says how far it looks is reported as
// looking at the least.
func (c Check[S]) At(strength Strength) Check[S] {
	c.Strength = strength
	return c
}

// Falsify lowers a row's falsifiability claim: a planted defect is the
// claim and the evidence in one field, an argument is the honest record
// that no defect can be constructed, and holding both is refused — a
// defect in hand beats an argument. Neither means unproven, which is what
// the report says about the row.
func Falsify(row string, hasDefect bool, argued string) (Falsifiability, error) {
	switch {
	case hasDefect && argued != "":
		return Falsifiability{}, fmt.Errorf(
			"check %q is both ProvenBy and Argued; a defect in hand beats an argument — drop Argued", row,
		)
	case hasDefect:
		return Proven(), nil
	case argued != "":
		return Argued(argued), nil
	default:
		return Falsifiability{}, nil
	}
}

// methodScoped refuses a row that set Method beside a body which fixes its
// own scope.
//
// The two disagree about what the check is called, and taking either would
// file it somewhere its author would not look for it. Refusing names the
// field to drop, because that is the shorter of the two edits.
func methodScoped(row, method string, scoped bool) error {
	if method == "" || scoped {
		return nil
	}
	return fmt.Errorf("check %q sets Method, but its body fixes its own scope; drop Method", row)
}
