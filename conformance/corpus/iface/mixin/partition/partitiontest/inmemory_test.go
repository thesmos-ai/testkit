// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `partition axis=partition read=Read` names the boundary and what observes it,
// and the model tier's law is that two partitions do not see each other. The
// row below is what one partition settles.
package partitiontest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/partition"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/partition/partitiontest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	partitiontest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	partitiontest.RunMixed(t,
		inMemory("in-memory"),
		partitiontest.MixedSuite.Without(partitiontest.MixedSuite.Checks.Put.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	partitiontest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) partitiontest.MixedHarness[*partitiontest.InMemory] {
	return partitiontest.MixedHarness[*partitiontest.InMemory]{
		Name: name, New: partitiontest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = partitiontest.MixedChecks{
	{
		Method: "Read", Name: "misses-a-key-its-partition-lacks",
		Claim: "Read reports a key its partition does not hold",
		Run:   missesAKeyItsPartitionLacks,
		ProvenBy: partitiontest.BrokenMixed(
			"a store that answers for any key in a partition it knows", newAnswersAnyKey,
		),
		ProvenReason: "an unwritten key is a miss",
	},
}

// --- Bodies -------------------------------------------------------------------

// missesAKeyItsPartitionLacks writes the partition first, so the miss below is
// about the KEY rather than about an empty store: a store answering for a key
// nobody wrote is indistinguishable from one that found it.
func missesAKeyItsPartitionLacks(
	tb testing.TB, s partition.Mixed, fx partitiontest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Put(tb.Context(), fx.Partition(), fx.Key(), fx.Value()),
		"the key is written into its partition")

	got, err := s.Read(tb.Context(), fx.Partition(), fx.Key())
	testkit.NoError(tb, err, "and reads back from it")
	testkit.Equal(tb, got, fx.Value(), "carrying what was written")

	_, err = s.Read(tb.Context(), fx.Partition(), fx.KeyOther())
	testkit.Error(tb, err, "an unwritten key is a miss")
}

// --- Planted defects ----------------------------------------------------------

// answersAnyKey remembers which partitions it has seen and answers for every
// key inside them, which is a lookup that checked the partition and forgot the
// key. The write and the read-back both succeed, so only the miss catches it.
type answersAnyKey struct{ seen map[string]string }

func newAnswersAnyKey() *answersAnyKey { return &answersAnyKey{seen: map[string]string{}} }

func (a *answersAnyKey) Put(_ context.Context, part, _, value string) error {
	a.seen[part] = value
	return nil
}

func (a *answersAnyKey) Read(_ context.Context, part, _ string) (string, error) {
	value, known := a.seen[part]
	if !known {
		return "", partitiontest.ErrNotFound
	}
	return value, nil
}
