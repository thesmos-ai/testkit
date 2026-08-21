// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal for the reason derive_laws_test.go is: the fixture populates
// the unexported stamp projection through the real key on a real bag,
// which only this package's own constructors reach.
package suite

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// The zero-on-error check induces its error from what the declaration
// says can go wrong.
//
// The reading that looks right and is not: "the method takes an input,
// so draw one that misses". The bus's Subscribe takes a topic and
// declares no miss, so an unsubscribed topic answers normally and a
// check drawing one would skip on every run — proving nothing while
// reading as coverage. A declared `notfound=` sentinel is what makes a
// miss an error, and only then is the alternate draw the error source.
func TestZeroOnErrorPicksItsErrorSource(t *testing.T) {
	t.Parallel()

	t.Run("a declared miss draws the alternate member", func(t *testing.T) {
		t.Parallel()

		iface := zeroIface(zeroReader(true))
		plans, _ := Signature{}.Derive(iface)
		body := zeroBodyOf(t, plans)
		miss, ok := body.(projection.ZeroOnMiss)
		testkit.True(t, ok, "a reader whose miss is declared inspects a miss")
		testkit.Equal(t, miss.Call.Args[1], projection.FixtureCall(projection.ExprFixture, "KeyOther"),
			"drawn from the alternate member, which nothing wrote")
		testkit.True(t, miss.Pool != "", "and the skip names the pool that would seed it")
	})

	t.Run("an undeclared miss cancels a context instead", func(t *testing.T) {
		t.Parallel()

		iface := zeroIface(zeroReader(false))
		plans, _ := Signature{}.Derive(iface)
		body := zeroBodyOf(t, plans)
		_, ok := body.(projection.ZeroOnCancel)
		testkit.True(t, ok,
			"nothing declares this reader's miss, so an unwritten input answers normally")
	})
}

// zeroReader is a keyed reader answering a value beside an error,
// declaring its miss sentinel or not.
func zeroReader(declaresMiss bool) subject.Method {
	m := subject.Method{
		Sig: &golang.Sig{
			Name: "Get",
			Params: []golang.Param{
				{Name: "ctx", Source: storefixture.PkgNamed("context", "Context")},
				{Name: "key", Field: "Key", Source: storefixture.Named("string")},
			},
			Returns: []golang.Return{
				{Source: storefixture.Named("Value")},
				{Error: true},
			},
		},
		ArgFields: []string{"Key"},
	}
	if !declaresMiss {
		return m
	}
	bag := sdk.NewBag()
	shape.MixinParamKey(MixinTTL, MixinTTLNotFound).Set(bag, "kv.ErrNotFound", "test")
	src := &node.Method{Name: m.Name}
	src.MetaBag = bag
	m.Source = src
	m.Mixins = []string{MixinTTL}
	m.MixinParams = mixinParamsOf(bag, m.Mixins)
	return m
}

// zeroIface pairs the method with a fixture that can deliver its draw.
func zeroIface(m subject.Method) Iface {
	return Iface{
		Name: "Store", Token: "store", Qualifier: "store",
		Methods: []subject.Method{m},
		Fixture: subject.Fixture{Fields: []subject.FixtureField{{
			Name:   "Key",
			Sample: golang.Sample{Text: `"k"`},
			Other:  golang.Sample{Text: `"o"`},
		}}},
	}
}

// zeroBodyOf selects the zero-on-error plan's body from a derived set.
func zeroBodyOf(t *testing.T, plans []projection.CheckPlan) projection.Body {
	t.Helper()
	for _, p := range plans {
		if p.ID.Seg == vocab.SegZeroValue {
			return p.Body
		}
	}
	t.Fatal("the signature rules license a zero-on-error check for this shape")
	return nil
}
