// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/fault"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// faultSymOf is the sentinel the crash schedule induces the subject's
// writer to fail with, nil where the declaration names none.
//
// Read off the `//testkit:fault` stamp on the writer, which is where a
// consumer says which errors that method can be made to report. The
// first is taken: a method naming several names them in the order its
// own declaration does, and the schedule needs one failure state rather
// than a survey of them.
//
// The identity is what matters and not the wording. The runner matches a
// subject's trigger on this exact error, and the claim the schedule
// states is about what the subject wrote down before reporting it.
func faultSymOf(pkg string, writer *subject.Method) *sdk.Expr {
	if writer == nil || writer.Source == nil {
		return nil
	}
	names := fault.Sentinels(writer.Source.Meta())
	if len(names) == 0 {
		return nil
	}
	return sdk.NewExternal(pkg, names[0])
}

// Faulted reports whether this interface's crash claim can be stated
// with the medium free to fail.
//
// The pair has to derive, and the writer has to name a sentinel it can
// be made to report. Without the second there is no failure to induce —
// and inducing is the whole design: a double answering an error in front
// of the subject never delivers the write, so the subject cannot have
// left anything behind and the claim has nothing to be false about.
func (b *Bindings) Faulted() bool { return b.Sim() && b.FaultSym != nil }
