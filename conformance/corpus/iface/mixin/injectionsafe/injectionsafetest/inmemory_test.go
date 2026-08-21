// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `injectionsafe` is the model tier's: what it claims is that no input is ever
// read as syntax, which needs a generated corpus of hostile ones. The row below
// carries the single case a consumer can name.
package injectionsafetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/injectionsafe"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/injectionsafe/injectionsafetest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	injectionsafetest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	injectionsafetest.RunMixed(t,
		inMemory("in-memory"),
		injectionsafetest.MixedSuite.Without(injectionsafetest.MixedSuite.Checks.Store.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	injectionsafetest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) injectionsafetest.MixedHarness[*injectionsafetest.InMemory] {
	return injectionsafetest.MixedHarness[*injectionsafetest.InMemory]{
		Name: name, New: injectionsafetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = injectionsafetest.MixedChecks{
	{
		Method: "Store", Name: "control-sequence-is-data",
		Claim: "Store round-trips a control sequence as data",
		Run:   controlSequenceIsData,
		ProvenBy: injectionsafetest.BrokenMixed(
			"a store that strips what it does not like the look of", newSanitises,
		),
		ProvenReason: "the value is data, not syntax",
	},
}

// --- Bodies -------------------------------------------------------------------

// controlSequenceIsData round-trips one hostile value. What makes it a check
// rather than a smoke call is the comparison: a store that accepted the value
// and mangled it satisfies "storing succeeds" and loses the data.
func controlSequenceIsData(
	tb testing.TB, s injectionsafe.Mixed, fx injectionsafetest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Store(tb.Context(), fx.Key(), hostile), "storing succeeds")

	got, err := s.Load(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "loading succeeds")
	testkit.Equal(tb, got, hostile, "the value is data, not syntax")
}

// --- Planted defects ----------------------------------------------------------

// hostile is the value the row round-trips: a fragment that is syntax to a
// naive backend and data to a correct one.
const hostile = `'; DROP TABLE users; --`

// sanitises strips the characters it distrusts on the way in, which is the
// wrong answer that looks like the right one: nothing is ever executed, and
// nothing the caller stored comes back either.
type sanitises struct{ held map[string]string }

func newSanitises() *sanitises { return &sanitises{held: map[string]string{}} }

func (s *sanitises) Store(_ context.Context, key, value string) error {
	cleaned := make([]rune, 0, len(value))
	for _, r := range value {
		if r != '\'' && r != ';' && r != '-' {
			cleaned = append(cleaned, r)
		}
	}
	s.held[key] = string(cleaned)
	return nil
}

func (s *sanitises) Load(_ context.Context, key string) (string, error) {
	value, held := s.held[key]
	if !held {
		return "", injectionsafetest.ErrMissing
	}
	return value, nil
}

// TestMixedLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestMixedLawsCanSaturate(t *testing.T) {
	t.Parallel()

	injectionsafetest.MixedModelSaturation(t, func() injectionsafetest.Mixed { return injectionsafetest.NewInMemory() })
}
