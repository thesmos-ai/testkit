// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal because the derivation and its listing are the seam that has
// to agree with itself, and both are unexported.
package model

import (
	"testing"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// A sugared field exists for a method this tier can already draw an
// argument for, out of a pool this run declares.
//
// Two conditions and each one has bitten. A pool that is not one of the
// shared pair leaves a field whose only honest body ignores its own
// parameter. A pool the run does not DECLARE leaves a draw expression
// calling a function nobody emitted, which is a compile error over a
// file the consumer may not edit.
func TestASugarNeedsAPoolTheRunDeclares(t *testing.T) {
	t.Parallel()

	str := sdk.Builtin("string")
	drawing := func(pool string, ref sdk.Ref) *Bindings {
		return &Bindings{
			Subject: subject.Subject{IfaceName: "Mixed"},
			Actions: []*Action{{Method: "Get", Pool: pool, Key: ref, Value: ref}},
		}
	}

	t.Run("a keyed reader gets one", func(t *testing.T) {
		t.Parallel()
		got := sugarsFor(drawing(poolKeys, str))
		testkit.Len(t, got, 1, "the method draws a key, and the run declares the key pool")
		testkit.Equal(t, got[0].Field, "PropGet", "named for the method it fixes the scope to")
		testkit.Equal(t, got[0].Label, "key", "and the draw is labelled in a shrunk counterexample")
	})

	t.Run("a method drawing from no shared pool gets none", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, sugarsFor(drawing("", str)), 0,
			"a field whose only honest body ignores its own parameter is worse than none")
	})

	t.Run("a method whose type nothing named gets none", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject: subject.Subject{IfaceName: "Mixed"},
			Actions: []*Action{{Method: "Get", Pool: poolKeys}},
		}
		testkit.Len(t, sugarsFor(b), 0,
			"a parameter with no type does not compile, so the sugar is not offered")
	})
}

// The listing OneBody offers names exactly the fields rendered beside
// it.
//
// One derivation for both, because a message offering PropPut to a row
// with no PropPut field sends a reader looking for something that is not
// there — and two derivations would drift the first time a sugar is
// added.
func TestTheBodyListingNamesTheFieldsBesideIt(t *testing.T) {
	t.Parallel()

	p := prop{Sugars: []PropSugar{{Field: "PropPut"}, {Field: "PropGet"}}}
	testkit.Equal(t, p.Fields(), ", Prop, PropPut, PropGet",
		"the un-sugared body first, then every sugar, appended to the harness generator's own")

	bare := prop{}
	testkit.Equal(t, bare.Fields(), ", Prop",
		"an interface with no drawable method still offers the un-sugared body")
}
