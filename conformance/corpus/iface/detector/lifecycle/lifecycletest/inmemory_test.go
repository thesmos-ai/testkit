// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Close carries the idempotent mixin, so Close/idempotent is derived and the
// row below is the same claim written by hand — kept as the worked example of a
// row standing beside the generated check it duplicates.
package lifecycletest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/lifecycle"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/lifecycle/lifecycletest"
)

// TestLifecycleContract runs the generated checks and this package's own.
func TestLifecycleContract(t *testing.T) {
	t.Parallel()

	lifecycletest.RunLifecycle(t, inMemory("in-memory"), lifecycleChecks)
}

// TestLifecycleContractWithoutSmoke drops a check through the typed index
// rather than a string, so a check that is renamed or stops being emitted
// breaks this compile instead of silently declining nothing.
func TestLifecycleContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	lifecycletest.RunLifecycle(t,
		inMemory("in-memory"),
		lifecycletest.LifecycleSuite.Without(lifecycletest.LifecycleSuite.Checks.Close.Smoke()),
	)
}

// TestLifecycleChecksCanFail drives the row against its planted defect — and
// against the same defect the generated Close/idempotent check is proven with,
// which is what makes "the same claim written by hand" verifiable rather than
// asserted in a comment.
func TestLifecycleChecksCanFail(t *testing.T) {
	t.Parallel()

	lifecycletest.ProveLifecycle(t, lifecycleChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) lifecycletest.LifecycleHarness[*lifecycletest.InMemory] {
	return lifecycletest.LifecycleHarness[*lifecycletest.InMemory]{
		Name: name, New: lifecycletest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var lifecycleChecks = lifecycletest.LifecycleChecks{
	{
		Method: "Close", Name: "second-close-succeeds",
		Claim: "Close is idempotent",
		Run:   secondCloseSucceeds,
		ProvenBy: lifecycletest.BrokenLifecycle(
			"a subject that refuses to be closed twice", newStrictCloser,
		),
		ProvenReason: "and so does the second",
	},
}

// --- Bodies -------------------------------------------------------------------

func secondCloseSucceeds(
	tb testing.TB, s lifecycle.Lifecycle, _ lifecycletest.LifecycleFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Close(tb.Context()), "the first close succeeds")
	testkit.NoError(tb, s.Close(tb.Context()), "and so does the second")
}

// --- Planted defects ----------------------------------------------------------

// strictCloser treats shutdown as a state change rather than a state, which
// takes down every caller whose deferred Close runs twice.
type strictCloser struct{ closed bool }

func newStrictCloser() *strictCloser { return &strictCloser{} }

func (c *strictCloser) Close(context.Context) error {
	if c.closed {
		return lifecycle.ErrClosed
	}
	c.closed = true
	return nil
}
