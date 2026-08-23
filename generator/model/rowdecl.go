// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/suite"
)

// KindRowDecl is the emit kind and template name for the function this
// tier's contributed expression names.
const KindRowDecl sdk.Kind = "model.rows.decl"

// RowDecl is the function the run surface's appended expression calls,
// and the rows it returns.
//
// The other half of [RowCall]. An expression naming a function nobody
// declared is a compile error over generated code a consumer cannot
// edit, so the two are contributed together, into the region for
// expressions and the region for declarations.
type RowDecl struct {
	sdk.BaseEmit

	// Func is the identifier, matching the call's.
	Func string

	// FixtureParam is the parameter, empty where the rows draw nothing;
	// FixtureType is its type.
	FixtureParam string
	FixtureType  string

	// Subject is the interface as the harness file spells it, for the
	// slice's element type.
	Subject string

	// TypeParams is the declaration's type-parameter list, empty for a
	// concrete interface.
	//
	// A generic subject spells its parameters at every use — the fixture
	// type, the check's element type — and a function using them has to
	// declare them. Rendered as the harness generator rendered its own,
	// because the two functions sit in one file and take the same
	// arguments.
	TypeParams []*sdk.EmitTypeParam

	// Vocab is the runtime suite package, whose Check the rows are.
	Vocab string

	// IndexVar is the typed index the harness generator emits. A row names
	// its identity through the index rather than composing one, so a
	// consumer's Without and the row report the same ID by construction —
	// and WHICH field of the index a row is named under rides on the row,
	// because this tier plans into more than one family.
	IndexVar string

	// LawIDs is the law catalogue's import path, for the Binds column: a
	// row names a law through the constant that declares it.
	LawIDs string

	// Rows is what the function returns.
	Rows []CheckRow
}

// Kind returns the template this contribution renders through.
func (*RowDecl) Kind() sdk.Kind { return KindRowDecl }

// rowDeclFor is the function [rowCallFor]'s expression names.
//
// Built from the same bindings and the same rows, so the call and the
// declaration cannot disagree about the identifier, the parameter or
// what comes back.
func rowDeclFor(
	c *sdk.Provenance, iface *sdk.Interface, b *Bindings, harness *suite.Contract,
) *RowDecl {
	d := &RowDecl{
		BaseEmit:   sdk.EmitBase(c, iface),
		Func:       b.RowsFuncName(),
		Subject:    harness.SubjectType(),
		Vocab:      VocabPkg,
		LawIDs:     suite.LawIDs,
		IndexVar:   projection.IndexVar(projection.Token(b.IfaceName)),
		Rows:       CheckRows(projection.Token(b.IfaceName), b.Rows),
		TypeParams: harness.TypeParams,
	}
	if drawsFixture(b, harness) {
		d.FixtureParam = fixtureIdent
		d.FixtureType = harness.Fixture.TypeName + harness.TypeArgs
	}
	return d
}
