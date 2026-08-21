// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/naming"
	"go.thesmos.sh/testkit/generator/suite"
)

// Every region the vocabulary names is one this host actually hands out.
//
// The failure this catches renders nothing and reports nothing. A name
// the host's switch does not know gets a fresh, empty region: the
// template ranges over no items, the contribution lands in a slot nobody
// reads, and the generated file comes out without it — which compiles.
// It cost a corpus regeneration to notice the first time.
func TestEveryRegionIsReachable(t *testing.T) {
	t.Parallel()

	var c suite.Contract
	for _, name := range naming.Slots() {
		reached := c.Slot(name)
		testkit.Equal(t, reached.SlotName, name, "the host returns the region asked for")
		testkit.True(t, reached.Owner != nil,
			"region "+name+" has no owner, so the host is minting a fresh one "+
				"and a contribution into it renders nothing")
	}
}

// The same region comes back on every reach, or a contribution and the
// template that renders it would be looking at different slices.
func TestARegionIsOneRegion(t *testing.T) {
	t.Parallel()

	var c suite.Contract
	for _, name := range naming.Slots() {
		first, second := c.Slot(name), c.Slot(name)
		testkit.True(t, first == second,
			"region "+name+" is minted anew on each reach, so a contribution "+
				"and the template that renders it read different slices")
	}
}
