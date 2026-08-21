// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

func TestDefectKinds(t *testing.T) {
	t.Parallel()

	t.Run("unique", func(t *testing.T) {
		t.Parallel()
		seen := map[projection.DefectKind]bool{}
		for _, k := range projection.DefectKinds() {
			testkit.False(t, seen[k], "defect kind "+string(k)+" must register once")
			seen[k] = true
		}
	})

	t.Run("dispatch-prefixed", func(t *testing.T) {
		t.Parallel()
		for _, k := range projection.DefectKinds() {
			testkit.HasPrefix(
				t,
				string(k),
				projection.DefectKindPrefix,
				"kinds are template names in the dispatch namespace",
			)
		}
	})
}
