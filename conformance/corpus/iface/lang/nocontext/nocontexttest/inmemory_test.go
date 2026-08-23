// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// An interface taking no context earns a smoke call per method and nothing
// else: cancellation, deadline and nil-context are claims about a parameter it
// does not have, and emitting them would not compile.
//
// So almost everything worth checking here is the author's, which is what the
// row table is for. The zero divisor comes from the row rather than the
// fixture: a generator derives plausible integers and every plausible divisor
// divides, so the error path is one derivation cannot reach.
package nocontexttest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/nocontext"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/nocontext/nocontexttest"
)

// TestCalculatorContract runs the generated checks and this package's own.
func TestCalculatorContract(t *testing.T) {
	t.Parallel()

	nocontexttest.RunCalculator(t, inMemory("in-memory"), calculatorChecks)
}

// TestCalculatorContractSuppression drops a check against the same subject:
// what is under test is the harness declining what it was told to, not the
// implementation.
func TestCalculatorContractSuppression(t *testing.T) {
	t.Parallel()

	nocontexttest.RunCalculator(t,
		inMemory("in-memory"),
		nocontexttest.CalculatorSuite.Without(nocontexttest.CalculatorSuite.Checks.Add.Smoke()),
	)
}

// TestCalculatorChecksCanFail drives every row against its planted defect.
func TestCalculatorChecksCanFail(t *testing.T) {
	t.Parallel()

	nocontexttest.ProveCalculator(t, inMemory("in-memory"), calculatorChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) nocontexttest.CalculatorHarness[*nocontexttest.InMemory] {
	return nocontexttest.CalculatorHarness[*nocontexttest.InMemory]{
		Name: name, New: nocontexttest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var calculatorChecks = nocontexttest.CalculatorChecks{
	{
		Method: "Add", Name: "is-commutative",
		Claim: "Add is commutative",
		Run:   isCommutative,
		ProvenBy: nocontexttest.BrokenCalculator(
			"a calculator that weights its first operand", planted(favoursTheFirstOperand),
		),
		ProvenReason: "does not depend on the order of its operands",
	},

	{
		Method: "Divide", Name: "reports-a-zero-divisor",
		Claim: "Divide reports a zero divisor",
		Run:   reportsAZeroDivisor,
		ProvenBy: nocontexttest.BrokenCalculator(
			"a calculator that answers zero rather than saying why",
			planted(swallowsTheZeroDivisor),
		),
		ProvenReason: "a zero divisor is an error rather than a panic",
	},
}

// --- Bodies -------------------------------------------------------------------

// isCommutative draws A and BOther rather than A and B.
//
// Both parameters are plain integers with no role, so the generator derives the
// same literal for each: A() and B() are both 1, and swapping them compares a
// call with itself. The companion value is what the fixture provides for
// exactly this — its own docblock says a second value is what lets a
// comparison mean something.
func isCommutative(
	tb testing.TB, s nocontext.Calculator, fx nocontexttest.CalculatorFixture,
) {
	tb.Helper()
	testkit.Equal(tb, s.Add(fx.A(), fx.BOther()), s.Add(fx.BOther(), fx.A()),
		"addition does not depend on the order of its operands")
}

func reportsAZeroDivisor(
	tb testing.TB, s nocontext.Calculator, fx nocontexttest.CalculatorFixture,
) {
	tb.Helper()
	_, err := s.Divide(fx.A(), 0)
	testkit.ErrorIs(tb, err, nocontexttest.ErrDivideByZero,
		"a zero divisor is an error rather than a panic")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted calculator gets wrong.
type fault int

const (
	// favoursTheFirstOperand weights the operands unequally, which is the
	// one arithmetic mistake a commutativity check exists to catch and a
	// smoke call cannot.
	favoursTheFirstOperand fault = iota

	// swallowsTheZeroDivisor answers zero with no error, which a caller
	// cannot tell from a real quotient of zero.
	swallowsTheZeroDivisor
)

// planted builds the constructor for one broken calculator.
func planted(wrong fault) func() plantedCalculator {
	return func() plantedCalculator { return plantedCalculator{wrong: wrong} }
}

type plantedCalculator struct{ wrong fault }

func (p plantedCalculator) Add(a, b int) int {
	if p.wrong == favoursTheFirstOperand {
		return 2*a + b
	}
	return a + b
}

func (p plantedCalculator) Divide(a, b int) (int, error) {
	if b == 0 {
		if p.wrong == swallowsTheZeroDivisor {
			return 0, nil
		}
		return 0, nocontexttest.ErrDivideByZero
	}
	return a / b, nil
}

// Reset is on the interface and no row is about it, so the defect keeps the
// only honest answer: there is nothing here to reset.
func (plantedCalculator) Reset() {}
