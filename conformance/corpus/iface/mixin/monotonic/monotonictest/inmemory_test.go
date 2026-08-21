// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package monotonictest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonic/monotonictest"
)

// monotonic is the model tier's — AUTO-MONOTONIC-NON-DECREASING states it — so
// the suite generates the signature family alone.
//
// Version takes nothing after its context, so it earns no zero-value check
// either: there is no input to choose that makes it fail.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	monotonictest.RunMixed(t,
		monotonictest.MixedHarness[*monotonictest.InMemory]{Name: "in-memory", New: monotonictest.NewInMemory},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	monotonictest.RunMixed(t,
		monotonictest.MixedHarness[*monotonictest.InMemory]{Name: "in-memory", New: monotonictest.NewInMemory},
		monotonictest.MixedSuite.Without(monotonictest.MixedSuite.Checks.Version.Smoke()),
	)
}
