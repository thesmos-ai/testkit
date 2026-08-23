// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sideeffecttest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sideeffect/sideeffecttest"
)

// The first generated check that calls a second method, and the first that
// could not be generated at all until eidos let the mixin name one.
//
// `//testkit:mixin sideeffect observe=Observed` is the whole of the wiring. The
// resolver rewrites Observed into a qualified name, the projection cuts it back
// to the local form a call site can spell, and the check observes either side of
// the call. Without the parameter the mixin said only *that* there was an effect
// — which is not a claim a test can make.
//
// So there is nothing hand-written here about the effect. The subject, the
// contract and one option is the whole file.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	sideeffecttest.RunMixed(t,
		sideeffecttest.MixedHarness[*sideeffecttest.InMemory]{Name: "in-memory", New: sideeffecttest.NewInMemory},
	)
}

// TestMixedChecksCanFail drives every planted defect through the check it
// is evidence for.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	sideeffecttest.ProveMixed(
		t,
		sideeffecttest.MixedHarness[*sideeffecttest.InMemory]{Name: "in-memory", New: sideeffecttest.NewInMemory},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	sideeffecttest.RunMixed(t,
		sideeffecttest.MixedHarness[*sideeffecttest.InMemory]{Name: "in-memory", New: sideeffecttest.NewInMemory},
		sideeffecttest.MixedSuite.Without(sideeffecttest.MixedSuite.Checks.Touch.Smoke()),
	)
}
