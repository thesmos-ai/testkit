// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package hookstest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/hooks/hookstest"
)

// The only generated check that constructs the thing it passes.
//
// A registration takes a callback, so the check has to build one — and the
// callback's own signature is what a func literal declares. It comes off the
// partner's func-typed parameter, spelled as types without names, which is all
// a literal needs and avoids inventing identifiers the body ignores.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	hookstest.RunMixed(t,
		hookstest.MixedHarness[*hookstest.InMemory]{Name: "in-memory", New: hookstest.NewInMemory},
	)
}

// TestMixedChecksCanFail drives every planted defect through the check it
// is evidence for.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	hookstest.ProveMixed(t, hookstest.MixedHarness[*hookstest.InMemory]{Name: "in-memory", New: hookstest.NewInMemory})
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	hookstest.RunMixed(t,
		hookstest.MixedHarness[*hookstest.InMemory]{Name: "in-memory", New: hookstest.NewInMemory},
		hookstest.MixedSuite.Without(hookstest.MixedSuite.Checks.Fire.Smoke()),
	)
}
