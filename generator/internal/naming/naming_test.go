// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package naming_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/builder"
	"go.thesmos.sh/testkit/generator/internal/naming"
	"go.thesmos.sh/testkit/generator/stub"
)

// The generator that emits a name and the generator that spells it get
// the same string, because both read this one.
//
// The failure this prevents is not a wrong name — it is HALF a rename.
// A double generator that renamed its suffix while the harness's proofs
// kept a literal would emit a companion naming a type that no longer
// exists, which fails in the consumer's compiler over generated code
// they cannot edit, attributed to the generator rather than to the
// rename. Both sides reading one constant is what makes that
// unrepresentable rather than merely unlikely.
func TestTheEmitterAndTheSpellerAgree(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, stub.DefaultSuffix, naming.StubSuffix,
		"the double generator's own suffix is this one")
	testkit.Equal(t, builder.CompanionSuffix, naming.CompanionSuffix,
		"and the builder's companion convention is this one")
}

// The conventions are trailers, so they compose onto a type name and
// nothing else.
func TestTheSuffixesAreTrailers(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, "KV"+naming.StubSuffix, "KVStub",
		"a double is the interface's name and this")
	testkit.Equal(t, "User"+naming.CompanionSuffix, "UserDefaults",
		"a seed function is the type's name and this — which is why it "+
			"carries the type rather than being a bare Defaults")
}
