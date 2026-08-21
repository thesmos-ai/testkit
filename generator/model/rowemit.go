// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"strings"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/suite"
)

// KindRowCall is the emit kind and template name for this tier's
// contribution to the run surface: the call that yields its checks.
const KindRowCall sdk.Kind = "model.rows.call"

// RowCall is the expression the run surface appends — one call to the
// function this tier emits, yielding every row it owns.
//
// It answers [suite.Rowed] as well as rendering, because the checks need
// naming in the typed index and the index is the harness generator's to
// emit. A consumer drops a check by naming it, so a check the index
// cannot name is one they cannot drop, prove, or run alone. This tier
// hands over the plans; that one names them.
type RowCall struct {
	sdk.BaseEmit

	// Func is the identifier of the function this tier emits.
	Func string

	// Fixture is the argument it takes, empty where the rows draw
	// nothing.
	Fixture string

	// Plans are the rows the call yields, in the order it yields them.
	// The index and the appended expression have to agree, and they agree
	// by both reading this.
	Plans []projection.CheckPlan
}

// Kind returns the template this contribution renders through.
func (*RowCall) Kind() sdk.Kind { return KindRowCall }

// CheckPlans implements [suite.Rowed].
func (r *RowCall) CheckPlans() []projection.CheckPlan { return r.Plans }

var _ suite.Rowed = (*RowCall)(nil)

// fixtureIdent is the fixture parameter's name in the run surface, where
// this tier's call is appended.
//
// Spelled to match what the harness generator calls it, because the call
// renders inside that generator's function and reads its parameter. A
// disagreement here is a compile error over generated code, which is the
// honest failure for a name two generators have to share.
const fixtureIdent = "fx"

// RowsFuncName is the function this tier emits and the contributed call
// names — `mixedModelRows`.
//
// Rows and not Checks: the harness generator's typed index already names
// a group type `<iface>ModelChecks`, and a function sharing that name
// reads as a conversion to it. Both land in one file, so the second
// naming is the one that has to move.
func (b *Bindings) RowsFuncName() string {
	return strings.ToLower(b.IfaceName[:1]) + b.IfaceName[1:] + "ModelRows"
}
