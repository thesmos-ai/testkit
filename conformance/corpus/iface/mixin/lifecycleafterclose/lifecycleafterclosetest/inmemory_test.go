// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The mixin's own claim, and one no single call can make: what changes between
// the two calls is the state, and a generated check meets a fresh subject.
package lifecycleafterclosetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/lifecycleafterclose"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/lifecycleafterclose/lifecycleafterclosetest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	lifecycleafterclosetest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	lifecycleafterclosetest.RunMixed(t,
		inMemory("in-memory"),
		lifecycleafterclosetest.MixedSuite.Without(lifecycleafterclosetest.MixedSuite.Checks.Close.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	lifecycleafterclosetest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) lifecycleafterclosetest.MixedHarness[*lifecycleafterclosetest.InMemory] {
	return lifecycleafterclosetest.MixedHarness[*lifecycleafterclosetest.InMemory]{
		Name: name, New: lifecycleafterclosetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = lifecycleafterclosetest.MixedChecks{
	{
		Method: "Work", Name: "refused-after-close",
		Claim: "Work refuses work after the subject closed",
		Run:   refusedAfterClose,
		ProvenBy: lifecycleafterclosetest.BrokenMixed(
			"a subject that keeps working after it closed", newWorksAfterClose,
		),
		ProvenReason: "refused rather than quietly done",
	},
}

// --- Bodies -------------------------------------------------------------------

func refusedAfterClose(
	tb testing.TB, s lifecycleafterclose.Mixed, _ lifecycleafterclosetest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Close(tb.Context()), "the subject closes")
	testkit.Error(tb, s.Work(tb.Context()),
		"and work after it is refused rather than quietly done")
}

// --- Planted defects ----------------------------------------------------------

// worksAfterClose closes cleanly and then does the work anyway, which is a
// teardown that released nothing — and the one thing a check meeting a fresh
// subject cannot observe.
type worksAfterClose struct{ closed bool }

func newWorksAfterClose() *worksAfterClose { return &worksAfterClose{} }

func (w *worksAfterClose) Close(context.Context) error {
	w.closed = true
	return nil
}

func (*worksAfterClose) Work(context.Context) error { return nil }
