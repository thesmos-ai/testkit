// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package multiargwritertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/multiargwriter/multiargwritertest"
)

// A three-argument write is a writer like any other.
//
// The seed derivation matched the single-argument `writer` alone, so the
// ordinary keyed store — the commonest write there is — could not populate its
// own subject and every consumer wrote the hook by hand. The three writer
// detectors differ in arity and nothing else, and the call passes whatever the
// method declares, so arity was never something it had to know.
func TestMultiArgWriterContract(t *testing.T) {
	t.Parallel()

	multiargwritertest.RunMultiArgWriter(
		t,
		multiargwritertest.MultiArgWriterHarness[*multiargwritertest.InMemory]{
			Name: "in-memory",
			New:  multiargwritertest.NewInMemory,
		},
	)
}

// TestMultiArgWriterChecksCanFail drives every planted defect through the check it
// is evidence for.
func TestMultiArgWriterChecksCanFail(t *testing.T) {
	t.Parallel()

	multiargwritertest.ProveMultiArgWriter(
		t,
		multiargwritertest.MultiArgWriterHarness[*multiargwritertest.InMemory]{
			Name: "in-memory",
			New:  multiargwritertest.NewInMemory,
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMultiArgWriterContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	multiargwritertest.RunMultiArgWriter(
		t,
		multiargwritertest.MultiArgWriterHarness[*multiargwritertest.InMemory]{
			Name: "in-memory",
			New:  multiargwritertest.NewInMemory,
		},
		multiargwritertest.MultiArgWriterSuite.Without(multiargwritertest.MultiArgWriterSuite.Checks.Set.Smoke()),
	)
}
