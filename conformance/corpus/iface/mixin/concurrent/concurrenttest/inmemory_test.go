// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package concurrenttest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/concurrent/concurrenttest"
)

// concurrent is the suite tier's under ADR-0018 and its check is still not
// generated, for the reason the RFC's open list records and the header now
// repeats as a refusal: concurrent callers not racing is observable only under
// the race detector, and `make check` runs `mod`, `lint`, `test`, `coverage`
// and `branch` — not `test race`.
//
// A generated check asserting nothing under the default gate would read as
// coverage while being decoration.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	concurrenttest.RunMixed(t,
		concurrenttest.MixedHarness[*concurrenttest.InMemory]{Name: "in-memory", New: concurrenttest.NewInMemory},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	concurrenttest.RunMixed(t,
		concurrenttest.MixedHarness[*concurrenttest.InMemory]{Name: "in-memory", New: concurrenttest.NewInMemory},
		concurrenttest.MixedSuite.Without(concurrenttest.MixedSuite.Checks.Bump.Smoke()),
	)
}

// TestMixedLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestMixedLawsCanSaturate(t *testing.T) {
	t.Parallel()

	concurrenttest.MixedModelSaturation(t, func() concurrenttest.Mixed { return concurrenttest.NewInMemory() })
}
