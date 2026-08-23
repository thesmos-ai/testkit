// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutatortest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/mutator/mutatortest"
)

// A write returning nothing earns two checks: the smoke call and nil-context
// tolerance. Cancellation and deadline are claims about what a method reports,
// and this one reports nothing.
//
// It is also excluded from the writer set the seed derivation reads, for
// exactly this reason — a seed through a void method cannot say whether it
// worked, which would leave every later check running against an empty subject
// and passing.
func TestMutatorContract(t *testing.T) {
	t.Parallel()

	mutatortest.RunMutator(t,
		mutatortest.MutatorHarness[*mutatortest.InMemory]{Name: "in-memory", New: mutatortest.NewInMemory},
	)
}

// TestMutatorChecksCanFail drives every planted defect through the check it
// is evidence for.
func TestMutatorChecksCanFail(t *testing.T) {
	t.Parallel()

	mutatortest.ProveMutator(
		t,
		mutatortest.MutatorHarness[*mutatortest.InMemory]{Name: "in-memory", New: mutatortest.NewInMemory},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMutatorContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	mutatortest.RunMutator(t,
		mutatortest.MutatorHarness[*mutatortest.InMemory]{Name: "in-memory", New: mutatortest.NewInMemory},
		mutatortest.MutatorSuite.Without(mutatortest.MutatorSuite.Checks.Touch.Smoke()),
	)
}
