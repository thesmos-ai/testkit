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

// TestMixedChecksCanFail drives every planted defect through the check it
// is evidence for.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	concurrenttest.ProveMixed(
		t,
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
