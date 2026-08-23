// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
)

// consumerText concatenates a fixture's hand-written test files — every
// non-generated *_test.go under its directory tree, which is where a corpus
// consumer arms a generated option. Generated files are excluded so a
// falsification companion mentioning an option does not count as arming it.
func consumerText(t *testing.T, dir string) string {
	t.Helper()
	var sb strings.Builder
	base := filepath.Join(corpusRoot, "corpus", dir)
	err := filepath.WalkDir(base,
		func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := d.Name()
			if d.IsDir() || !strings.HasSuffix(name, "_test.go") || strings.Contains(name, ".gen") {
				return nil
			}
			// The corpus is this repository's own checkout, not untrusted
			// input; the walk cannot race a symlink nobody plants.
			body, rErr := os.ReadFile(path) //nolint:gosec // see above
			if rErr != nil {
				return fmt.Errorf("read %s: %w", path, rErr)
			}
			sb.Write(body)
			return nil
		})
	testkit.NoError(t, err, "the fixture's consumer tests are readable")
	return sb.String()
}

// TestEveryDoorIsArmedOrArgued is the unarmed-door census: a guarded law, a
// clocked law, or an undeclared optional role is a visible skip unless some
// consumer arms it or the register argues why not — and an argued row for a
// door that is now armed is a stale excuse. No counts anywhere: the door
// set derives from the emission, the arming from the consumers' own calls,
// and every failure names its fixture and law.
func TestEveryDoorIsArmedOrArgued(t *testing.T) {
	t.Parallel()

	census, err := censusOnce()
	emitted := census.Emitted
	testkit.NoError(t, err, "the emission census runs")

	seen := map[string]bool{}
	verdict := func(key, armed string, isArmed bool) {
		seen[key] = true
		reason, argued := gate.UnarmedDoors[key]
		switch {
		case isArmed && argued:
			t.Errorf("%s is armed by a consumer (%s) and still registered — "+
				"delete the stale row: %s", key, armed, reason)
		case !isArmed && !argued:
			t.Errorf("%s is neither armed by a consumer (no %s in the "+
				"fixture's tests) nor argued in gate.UnarmedDoors — a guarded "+
				"law nobody arms is a visible skip", key, armed)
		}
	}

	for _, e := range emitted {
		if e.Dir == "" {
			continue
		}
		tests := consumerText(t, e.Dir)
		for law, doors := range e.Doors {
			for _, door := range doors {
				// The key the harness answers under, quoted as the Provide
				// map spells it. It used to be a generated option per door;
				// the open half of the capability registry replaced that
				// with one map, so the door's own name is what a consumer
				// writes and what this looks for.
				arm := strconv.Quote(door) + ":"
				verdict(e.Dir+"/"+law+"."+door, arm, strings.Contains(tests, arm))
			}
		}
		for _, law := range e.Clocked {
			// A clock is per-instance, so it keeps a field of its own on
			// the harness rather than a row in the Provide map.
			const arm = "OnClock:"
			verdict(e.Dir+"/"+law+".clock", arm, strings.Contains(tests, arm))
		}
		for law, roles := range e.Unarmed {
			for _, role := range roles {
				// An undeclared optional role has no option to arm — the
				// declaration itself is the arming — so a row is always owed.
				verdict(e.Dir+"/"+law+"."+role, "redeliver= on the directive", false)
			}
		}
	}

	// Every stale row at once, for the reason the owed-law census reports
	// its whole set: a register cleaned one entry per run is a register
	// nobody can see the shape of.
	var stale []string
	for key := range gate.UnarmedDoors {
		if !seen[key] {
			stale = append(stale, key)
		}
	}
	slices.Sort(stale)
	testkit.Len(t, stale, 0, "registered but the emission produced no such door — "+
		"the rows outlived their debt: "+strings.Join(stale, ", "))
}

// TestStampedMissIdentityReachesTheSequences holds the declaration's routed
// miss sentinel to the sequences that compare it: a fixture whose stamp the
// derived oracle consumes must also arm the identity on its reader actions,
// or the comparison silently regressed to presence.
func TestStampedMissIdentityReachesTheSequences(t *testing.T) {
	t.Parallel()

	census, err := censusOnce()
	emitted := census.Emitted
	testkit.NoError(t, err, "the emission census runs")

	for _, e := range emitted {
		if e.SentinelStamped {
			testkit.True(t, e.SentinelArmed,
				e.Fixture+" stamps a miss sentinel the oracle routes, and no sequence carries it")
		}
	}
}
