// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package compositewritertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/compositewriter/compositewritertest"
)

// The generated contract, run against the in-memory subject. The fixture
// exists to pin the composite-writer detector — a key beside a value, an
// error and nothing else — and the identity gate holds the stamp to the
// directory's name, so the wiring here stays the minimal consumer's.
func TestCompositeWriterContract(t *testing.T) {
	t.Parallel()

	compositewritertest.RunCompositeWriter(t,
		compositewritertest.CompositeWriterHarness[*compositewritertest.InMemory]{
			Name: "in-memory", New: compositewritertest.NewInMemory,
		},
	)
}

// TestCompositeWriterChecksCanFail drives every planted defect through the check it
// is evidence for.
func TestCompositeWriterChecksCanFail(t *testing.T) {
	t.Parallel()

	compositewritertest.ProveCompositeWriter(
		t,
		compositewritertest.CompositeWriterHarness[*compositewritertest.InMemory]{
			Name: "in-memory",
			New:  compositewritertest.NewInMemory,
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestCompositeWriterContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	compositewritertest.RunCompositeWriter(t,
		compositewritertest.CompositeWriterHarness[*compositewritertest.InMemory]{
			Name: "in-memory", New: compositewritertest.NewInMemory,
		},
		compositewritertest.CompositeWriterSuite.Without(compositewritertest.CompositeWriterSuite.Checks.Set.Smoke()),
	)
}
