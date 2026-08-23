// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `errors` names more than one sentinel, and the row below is the second: the
// generated Get/miss check covers the absent key, and this covers the removed
// one. Telling them apart is the whole reason the mixin is worth having.
package errorstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/errors"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/errors/errorstest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	errorstest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	errorstest.RunMixed(t,
		inMemory("in-memory"),
		errorstest.MixedSuite.Without(errorstest.MixedSuite.Checks.Get.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	errorstest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) errorstest.MixedHarness[*errorstest.InMemory] {
	return errorstest.MixedHarness[*errorstest.InMemory]{
		Name: name, New: errorstest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = errorstest.MixedChecks{
	{
		Method: "Get", Name: "removed-differs-from-missing",
		Claim: "Get reports each declared sentinel for its own case",
		Run:   removedDiffersFromMissing,
		ProvenBy: errorstest.BrokenMixed(
			"a store that reports every absence the same way", newOneSentinelForAll,
		),
		ProvenReason: "distinguishable from a missing one",
	},
}

// --- Bodies -------------------------------------------------------------------

func removedDiffersFromMissing(
	tb testing.TB, s errors.Mixed, _ errorstest.MixedFixture,
) {
	tb.Helper()
	_, err := s.Get(tb.Context(), errorstest.GoneKey())
	testkit.ErrorIs(tb, err, errors.ErrGone,
		"a removed key is distinguishable from a missing one")
}

// --- Planted defects ----------------------------------------------------------

// oneSentinelForAll answers the same error for a key that was removed and one
// that never existed, which is the collapse this mixin exists to forbid — and
// which the generated miss check calls correct, because it only ever asks about
// the absent one.
type oneSentinelForAll struct{}

func newOneSentinelForAll() oneSentinelForAll { return oneSentinelForAll{} }

func (oneSentinelForAll) Get(_ context.Context, _ string) (string, error) {
	return "", errors.ErrNotFound
}
