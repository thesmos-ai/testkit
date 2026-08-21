// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package projection is the build-time plan both tiers compute into and
// render from: the blueprint a generation run produces and the templates
// read. Everything here is build-time — a CheckPlan lives for
// milliseconds inside `testkit run` and is never seen by a consumer; the
// generated file it describes constructs the runtime's own [suite.Check]
// values instead.
//
// One inventory sources every artifact. Claim text, probe sets, lock
// rows, the typed index, the proofs table and the selfcheck census are
// all projections of the same nodes, which is what makes "a claim is
// exactly as wide as its assertion" a property of the data flow rather
// than a rule enforced by review.
//
// # One derivation, many renderers
//
// Both tiers plan into this, and the harness generator computes it,
// because the harness is derived from the plans rather than from the
// stamps — see [HarnessOf]. A capability nothing checks is a field
// nobody may demand, so a clocked law's row and the Clock field it needs
// have to be worked out together. Splitting the planning by tier would
// give the model tier's clocked check a harness with no clock on it.
//
// What differs per tier is who RENDERS a row. A plan whose body kind no
// template in the computing tier spells is listed under Withheld, which
// is the honest state for a row planned here and emitted elsewhere.
//
// It lives under internal/ rather than inside the harness generator for
// the reason [go.thesmos.sh/testkit/generator/internal/subject] gives:
// the biggest consumer of a shared derivation is not its owner.
package projection

import (
	"errors"
	"fmt"

	"go.thesmos.sh/testkit/engine/suite"
)

// IDPlan spells one check identity in the grammar's terms rather than
// as a string, so the emitter cannot mint an ID shape the runtime
// would refuse. Exactly one of Method or Family is set: method-scoped
// IDs render "Method/seg", family-scoped IDs render
// "family/qualifier/seg" — and the qualifier is unconditional for
// family scopes, per the uniform-qualification ruling.
type IDPlan struct {
	Method    string // "Append" -> "Append/<seg>"
	Family    string // suite.FamilyModel etc. -> "<family>/<qualifier>/<seg>"
	Qualifier string // the interface token; required with Family
	Seg       string
}

// Render produces the runtime ID through the engine vocabulary — the
// one home of the grammar.
//
// A malformed plan is a deriver bug rather than consumer input, and it
// is still reported rather than panicked: [Inventory.Verify] is the
// seam that holds a run to its own invariants, and a panic here would
// jump straight over it — taking every other interface's output down
// for one deriver's mistake instead of failing the interface being
// derived, with the plan named.
func (p IDPlan) Render() (suite.ID, error) {
	switch {
	case p.Method != "" && p.Family != "":
		return "", fmt.Errorf("projection: ID plan sets both Method %q and Family %q", p.Method, p.Family)
	case p.Method != "":
		return suite.MethodID(p.Method, p.Seg), nil
	case p.Family != "":
		if p.Qualifier == "" {
			return "", fmt.Errorf(
				"projection: family ID %s/%s lacks its interface qualifier; qualification is unconditional",
				p.Family,
				p.Seg,
			)
		}
		return suite.FamilyID(p.Family, p.Qualifier, p.Seg), nil
	default:
		return "", errors.New("projection: empty ID plan")
	}
}

// CheckPlan is the blueprint for one emitted check. The closed Body
// and Defect variant sets are deliberate: a family that wants "one
// more optional field" adds a variant and a template instead, visibly,
// which is the guard against this node rotting into a god-struct.
type CheckPlan struct {
	ID    IDPlan
	Class suite.Class
	Claim string

	// Needs names the capability doors the runtime gate consults;
	// values are rendered into the check's Caps literal.
	Needs []NeedPlan

	// Body is how the check asserts — exactly one variant.
	Body Body

	// Falsifiable carries the claim about the claim: Proven demands a
	// Defect; Argued demands a reason and forbids one. The inventory's
	// census refuses any other combination.
	Falsifiable suite.Falsifiability

	// Defect is the planted implementation that must red this check —
	// nil exactly when the check is Argued.
	Defect Defect

	// Binds names the assertion bodies the check delegates to, and
	// renders into the lock's fourth column, which is what makes
	// narrowing a probe set diff.
	Binds []Bind

	// Licensed names the classification that bought this check, empty
	// for one the shape alone earned.
	//
	// Recorded here rather than read back off [Class], because the two
	// do not line up: a seeded reader's hit check is class `mixin/reader`
	// and `reader` is a DETECTOR, so a census taking the axis off the
	// class prefix files every detector's evidence under the wrong one.
	// The deriver knows which stamp it read; anything downstream would
	// be guessing from the answer.
	Licensed Licence
}

// The three classification axes, which are the three registries eidos
// ships and the three a census reads back.
//
// Spelled here rather than imported from the conformance gate: the gate
// is a different module and reads these, so taking its constants would
// point the arrow the wrong way — what a check was derived from is the
// generator's answer, and the census is a consumer of it.
const (
	AxisDetector = "detector"
	AxisMixin    = "mixin"
	AxisContract = "contract"
)

// Licence is the classification a check was derived from: which axis
// registers it, and its name on that axis.
//
// The pair rather than the name alone, because the three registries
// are separate namespaces and a bare name cannot be looked up in the
// right one — `pure` is both a detector shape and a mixin.
type Licence struct {
	Axis, Name string
}

// Licensed reports whether a classification bought this check.
func (l Licence) Licensed() bool { return l.Axis != "" && l.Name != "" }

// NeedPlan is one capability door: the runtime capability constant
// plus the rendered value literal, empty for presence-only doors.
type NeedPlan struct {
	Capability suite.Capability
	Value      Expr
}

// Body is the closed set of check-body shapes. One template per kind;
// the kind IS the template's name, and the census holds the two
// registries equal.
type Body interface {
	BodyKind() BodyKind

	// Strength is how far this body looks before passing. On the
	// interface rather than in a table beside it, so a variant added
	// without deciding does not compile — the same guard [BodyKind]
	// gets, for a property with the same failure mode if guessed.
	Strength() suite.Strength
}

// Defect is the closed set of planted-defect shapes, mirroring the
// proofs rules in derivation-rules.md.
type Defect interface{ DefectKind() DefectKind }
