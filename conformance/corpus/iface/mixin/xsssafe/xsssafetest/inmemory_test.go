// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `xsssafe` is the model tier's: that no input ever escapes as markup is a
// claim about a generated corpus of hostile ones. The row below carries the
// single case a consumer can name.
package xsssafetest_test

import (
	"context"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/xsssafe"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/xsssafe/xsssafetest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	xsssafetest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	xsssafetest.RunMixed(t,
		inMemory("in-memory"),
		xsssafetest.MixedSuite.Without(xsssafetest.MixedSuite.Checks.Render.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	xsssafetest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) xsssafetest.MixedHarness[*xsssafetest.InMemory] {
	return xsssafetest.MixedHarness[*xsssafetest.InMemory]{
		Name: name, New: xsssafetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = xsssafetest.MixedChecks{
	{
		Method: "Render", Name: "leaves-no-angle-bracket",
		Claim: "Render leaves no angle bracket in the output",
		Run:   leavesNoAngleBracket,
		ProvenBy: xsssafetest.BrokenMixed(
			"a renderer that escapes only the opening bracket", newEscapesTheOpener,
		),
		ProvenReason: "no bracket survives escaping",
	},
}

// --- Bodies -------------------------------------------------------------------

func leavesNoAngleBracket(
	tb testing.TB, s xsssafe.Mixed, _ xsssafetest.MixedFixture,
) {
	tb.Helper()
	got, err := s.Render(tb.Context(), hostile)
	testkit.NoError(tb, err, "rendering succeeds")
	testkit.NotContains(tb, got, "<", "no bracket survives escaping")
}

// --- Planted defects ----------------------------------------------------------

// hostile is the fragment the row renders: markup to a naive renderer and text
// to a correct one.
const hostile = `<script>alert(1)</script>`

// escapesTheOpener handles the character everybody remembers and leaves the
// closing bracket alone, which is the half-done escaping that passes review and
// still lets a tag through in the cases the corpus covers.
type escapesTheOpener struct{}

func newEscapesTheOpener() escapesTheOpener { return escapesTheOpener{} }

func (escapesTheOpener) Render(_ context.Context, in string) (string, error) {
	return strings.ReplaceAll(in, ">", "&gt;"), nil
}
