// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `codec fidelity=lossy` is the weaker claim, stated once: not the identity,
// but stability — whatever the fold lost stays lost, and re-encoding the
// recovery reproduces the first encoding.
package codeclossytest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	codeclossy "go.thesmos.sh/testkit/conformance/corpus/iface/contract/codec-lossy"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/codec-lossy/codeclossytest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	codeclossytest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	codeclossytest.RunContract(t,
		inMemory("in-memory"),
		codeclossytest.ContractSuite.Without(codeclossytest.ContractSuite.Checks.Encode.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	codeclossytest.ProveContract(t, contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) codeclossytest.ContractHarness[*codeclossytest.InMemory] {
	return codeclossytest.ContractHarness[*codeclossytest.InMemory]{
		Name: name, New: codeclossytest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = codeclossytest.ContractChecks{
	{
		Method: "Decode", Name: "second-pass-agrees",
		Claim: "Decode agrees with itself on the second pass",
		Run:   secondPassAgrees,
		ProvenBy: codeclossytest.BrokenContract(
			"a codec that loses something new on every pass", newLosesEachTime,
		),
		ProvenReason: "loses nothing new",
	},
}

// --- Bodies -------------------------------------------------------------------

func secondPassAgrees(
	tb testing.TB, s codeclossy.Contract, fx codeclossytest.ContractFixture,
) {
	tb.Helper()
	encoded, err := s.Encode(tb.Context(), fx.In())
	testkit.NoError(tb, err, "the forward transform succeeds")

	decoded, err := s.Decode(tb.Context(), encoded)
	testkit.NoError(tb, err, "the inverse recovers what survived")

	again, err := s.Encode(tb.Context(), decoded)
	testkit.NoError(tb, err, "a second pass still encodes")
	testkit.Equal(tb, again, encoded, "and loses nothing new")
}

// --- Planted defects ----------------------------------------------------------

// losesEachTime drops a character on every forward pass, which is lossy and not
// stable: each round trip degrades further, so a value re-encoded enough times
// becomes nothing. That is the difference `fidelity=lossy` still forbids.
type losesEachTime struct{}

func newLosesEachTime() losesEachTime { return losesEachTime{} }

func (losesEachTime) Encode(_ context.Context, in string) (string, error) {
	if in == "" {
		return "", nil
	}
	return in[:len(in)-1], nil
}

func (losesEachTime) Decode(_ context.Context, in string) (string, error) {
	return in, nil
}

// TestContractLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestContractLawsCanSaturate(t *testing.T) {
	t.Parallel()

	codeclossytest.ContractModelSaturation(t, func() codeclossytest.Contract { return codeclossytest.NewInMemory() })
}
