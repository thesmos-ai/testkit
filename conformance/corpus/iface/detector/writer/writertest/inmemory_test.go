// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package writertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/writer/writertest"
)

// The minimal consumer: a name and a constructor, and nothing else.
//
// What Put is handed is derived from Value's own fields, so no fixture is
// supplied and no row is written — this file is what a harness costs when the
// signature says everything.
func TestWriterContract(t *testing.T) {
	t.Parallel()

	writertest.RunWriter(t,
		writertest.WriterHarness[*writertest.InMemory]{Name: "in-memory", New: writertest.NewInMemory},
	)
}

// TestWriterChecksCanFail drives every planted defect through the check it
// is evidence for.
func TestWriterChecksCanFail(t *testing.T) {
	t.Parallel()

	writertest.ProveWriter(
		t,
		writertest.WriterHarness[*writertest.InMemory]{Name: "in-memory", New: writertest.NewInMemory},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestWriterContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	writertest.RunWriter(t,
		writertest.WriterHarness[*writertest.InMemory]{Name: "in-memory", New: writertest.NewInMemory},
		writertest.WriterSuite.Without(writertest.WriterSuite.Checks.Put.Smoke()),
	)
}
