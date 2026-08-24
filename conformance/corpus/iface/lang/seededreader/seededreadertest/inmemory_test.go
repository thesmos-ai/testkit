// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package seededreadertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/seededreader/seededreadertest"
)

// The seed seam end to end: the harness receives the corpus the run
// zipped from its own pools, and the generated hit and count checks read
// back exactly what was put in.
//
// Neither check is writable against an unseeded subject — a hit has
// nothing to hit and a count has no number to expect — which is why they
// derive only where a corpus exists.
func TestCatalogContract(t *testing.T) {
	t.Parallel()

	seededreadertest.RunCatalog(t,
		seededreadertest.CatalogHarness[*seededreadertest.InMemory]{
			Name: "in-memory",
			Seed: seededreadertest.NewInMemory,
		},
	)
}

// TestCatalogChecksCanFail drives every planted defect through the check it
// is evidence for.
func TestCatalogChecksCanFail(t *testing.T) {
	t.Parallel()

	seededreadertest.ProveCatalog(
		t,
		seededreadertest.CatalogHarness[*seededreadertest.InMemory]{
			Name: "in-memory",
			Seed: seededreadertest.NewInMemory,
		},
	)
}

// TestCatalogFromAStartedSeed builds the seeded subject through the
// harness's Start-shaped seed door.
//
// The same corpus, reached the other way. Seed is for a seeded
// constructor that cannot fail; StartSeed is for one needing the test's
// lifetime — an index to close, a directory to clear — and it is handed
// the testing.TB to register that on. Both doors ship and only one was
// ever opened.
func TestCatalogFromAStartedSeed(t *testing.T) {
	t.Parallel()

	seededreadertest.RunCatalog(t,
		seededreadertest.CatalogHarness[*seededreadertest.InMemory]{
			Name:      "in-memory from a started seed",
			StartSeed: startSeeded,
		},
	)
}

// startSeeded builds the catalogue over the run's corpus and registers
// its teardown, which is the whole difference from Seed.
func startSeeded(
	tb testing.TB, docs seededreadertest.CatalogCorpus,
) *seededreadertest.InMemory {
	tb.Helper()
	s := seededreadertest.NewInMemory(docs)
	tb.Cleanup(func() { _ = s })
	return s
}
