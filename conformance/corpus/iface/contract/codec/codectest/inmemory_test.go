// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `codec fidelity=exact` is the claim that the round trip is the identity — a
// lossy codec would declare the weaker form, and this assertion would be wrong
// for it. The pair is stated once below.
package codectest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/codec"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/codec/codectest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	codectest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	codectest.RunContract(t,
		inMemory("in-memory"),
		codectest.ContractSuite.Without(codectest.ContractSuite.Checks.Encode.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	codectest.ProveContract(t, contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) codectest.ContractHarness[*codectest.InMemory] {
	return codectest.ContractHarness[*codectest.InMemory]{
		Name: name, New: codectest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = codectest.ContractChecks{
	{
		Method: "Decode", Name: "undoes-encode",
		Claim: "Decode undoes exactly what Encode did",
		Run:   undoesEncode,
		ProvenBy: codectest.BrokenContract(
			"a codec whose inverse drops what the forward pass added", newLosesOnDecode,
		),
		ProvenReason: "exact fidelity is the identity",
	},
}

// --- Bodies -------------------------------------------------------------------

func undoesEncode(tb testing.TB, s codec.Contract, fx codectest.ContractFixture) {
	tb.Helper()
	encoded, err := s.Encode(tb.Context(), fx.In())
	testkit.NoError(tb, err, "the forward transform succeeds")

	decoded, err := s.Decode(tb.Context(), encoded)
	testkit.NoError(tb, err, "and the inverse undoes it")
	testkit.Equal(tb, decoded, fx.In(), "exact fidelity is the identity")
}

// --- Planted defects ----------------------------------------------------------

// losesOnDecode encodes reversibly and decodes to something shorter, which is
// what a codec with a trimming step on one side only does. Both calls succeed,
// so nothing but comparing the ends catches it.
type losesOnDecode struct{}

func newLosesOnDecode() losesOnDecode { return losesOnDecode{} }

func (losesOnDecode) Encode(_ context.Context, in string) (string, error) {
	return "enc:" + in, nil
}

func (losesOnDecode) Decode(_ context.Context, in string) (string, error) {
	return in, nil
}
