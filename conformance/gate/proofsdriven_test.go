// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
)

// TestEveryPlantedDefectIsDriven holds each fixture to running the proofs
// its own run surface declares.
//
// The generated companion used to drive them itself, and could not keep
// doing it: a check may need a capability, only the consumer's harness
// answers one, and a test function in the generated package has no way to
// reach a harness written in the consumer's. So the defect map moved to
// ProveX, which takes the harness — and with it went the one thing that
// guaranteed the proofs ran at all.
//
// This is what replaced that guarantee for the corpus. A package whose
// checks claim Proven and whose consumer never calls ProveX reports every
// one of those claims in its census and tests none of them, which is the
// exact reading the whole falsifiability layer exists to prevent.
//
// It measures the files rather than the emission because that is where the
// absence lives: the emission says a defect was derived, and only the
// consumer's own test says anyone drives it.
func TestEveryPlantedDefectIsDriven(t *testing.T) {
	t.Parallel()

	var undriven []string
	dirs := packagesDeclaringProofs(t)
	testkit.True(t, len(dirs) > 0, "the corpus emits planted defects at all")
	for _, dir := range dirs {
		if !strings.Contains(consumerText(t, dir), "Prove") {
			undriven = append(undriven, dir)
		}
	}
	slices.Sort(undriven)
	testkit.Len(t, undriven, 0, "declares planted defects and no consumer test drives "+
		"them — call ProveX with the harness: "+strings.Join(undriven, ", "))
}

// packagesDeclaringProofs is every corpus directory whose run surface holds
// a defect map, as the corpus-relative path the consumer census keys on.
func packagesDeclaringProofs(t *testing.T) []string {
	t.Helper()
	base := filepath.Join(corpusRoot, "corpus")
	var out []string
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "iface_suite.gen.go" {
			return err
		}
		// The corpus is this repository's own checkout, not untrusted input.
		body, rErr := os.ReadFile(path) //nolint:gosec // see above
		if rErr != nil {
			return fmt.Errorf("read %s: %w", path, rErr)
		}
		// The map's own declaration, which the run surface emits only where
		// this run derived a defect it could spell.
		if !strings.Contains(string(body), "Proofs() prove.Defects[") {
			return nil
		}
		rel, rErr := filepath.Rel(base, filepath.Dir(filepath.Dir(path)))
		if rErr != nil {
			return fmt.Errorf("locate %s under the corpus: %w", path, rErr)
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	testkit.NoError(t, err, "the corpus's generated run surfaces are readable")
	return out
}
