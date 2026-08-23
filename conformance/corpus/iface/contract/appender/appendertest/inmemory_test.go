// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package appendertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/appender/appendertest"
)

// appender is the model tier's under ADR-0018: `AUTO-APPEND-ONLY-GROWS` states
// it, and the suite tier implements no property a law already carries.
//
// So what the harness generates here is the signature-derived family, and it is
// not nothing — a log that panicked on a derived value, or applied a write for a
// cancelled caller, fails before any law runs.
func TestContractContract(t *testing.T) {
	t.Parallel()

	appendertest.RunContract(t,
		appendertest.ContractHarness[*appendertest.InMemory]{Name: "in-memory", New: appendertest.NewInMemory},
	)
}

// TestContractChecksCanFail drives every planted defect through the check it
// is evidence for.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	appendertest.ProveContract(
		t,
		appendertest.ContractHarness[*appendertest.InMemory]{Name: "in-memory", New: appendertest.NewInMemory},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	appendertest.RunContract(t,
		appendertest.ContractHarness[*appendertest.InMemory]{Name: "in-memory", New: appendertest.NewInMemory},
		appendertest.ContractSuite.Without(appendertest.ContractSuite.Checks.Run.Smoke()),
	)
}
