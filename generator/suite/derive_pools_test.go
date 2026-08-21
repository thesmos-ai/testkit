// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal for the reason derive_stamps_test.go is: the fixtures
// stamp real bags through the real annotators, and the walk under
// test is the unexported projection the shell calls.
package suite

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/defaults"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
	"go.thesmos.sh/testkit/generator/roles"
)

const poolFixtureFile = "kv/kv.go"

// resolveMap is the smallest [golang.Resolver]: fixture struct
// declarations keyed by name.
type resolveMap map[string]node.Node

func (r resolveMap) Resolve(t *node.TypeRef) (node.Node, bool) {
	if t == nil {
		return nil, false
	}
	n, held := r[t.Name]
	return n, held
}

// poolField declares one fixture field: a role (empty for unroled)
// and a default (empty for undeclared).
type poolField struct {
	name, role, stamp string
}

// poolFixture builds a request struct carrying the declared stamps
// through the real annotators, and answers the resolver plus a
// method drawing the struct.
func poolFixture(t *testing.T, fields ...poolField) (golang.Resolver, []subject.Method) {
	t.Helper()
	store := storefixture.New().
		Package("kv", "example.com/kv").
		File(poolFixtureFile, nil).
		Struct("PutRequest", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At(poolFixtureFile, 1, 1))
			for i, fld := range fields {
				b.Field(fld.name, storefixture.Named("string"), func(f *storefixture.FieldBuilder) {
					f.Pos(sdk.At(poolFixtureFile, i+2, 1))
					if fld.role != "" {
						f.Directive(storefixture.Directive("role", storefixture.Arg(fld.role)))
					}
					if fld.stamp != "" {
						f.Directive(storefixture.Directive("default", storefixture.Arg(fld.stamp)))
					}
				})
			}
		}).
		Build()
	plugintest.Annotate(t, roles.New(), store)
	plugintest.Annotate(t, defaults.New(), store)

	resolver := resolveMap{}
	for _, s := range store.Nodes().Structs().Items() {
		resolver[s.Name] = s
	}
	method := subject.Method{Sig: &golang.Sig{
		Name:   "Put",
		Params: []golang.Param{{Name: "req", Source: storefixture.Named("PutRequest")}},
	}}
	return resolver, []subject.Method{method}
}

func TestPoolsDeriveFromRoledDefaults(t *testing.T) {
	t.Parallel()

	r, methods := poolFixture(t,
		poolField{"Key", "key", `"test-key"`},
		poolField{"Value", "payload", `Value{Body: "test-body"}`},
		poolField{"TTL", "", `time.Minute`},
	)
	pools, refusals := poolsOf(r, methods)
	testkit.Len(t, refusals, 0, "roled, defaulted fields refuse nothing")
	testkit.Len(t, pools, 2, "one pool per roled field; the unroled TTL pins config instead")

	testkit.Equal(t, pools[0], projection.PoolPlan{
		Role:  "key",
		Field: "KeyPool",
		Members: [3]projection.Expr{
			`"test-key"`, `"other-key"`, `"\x00hostile\xffkey"`,
		},
		Type: golang.RefFor("string", ""),
	}, "the key pool, member for member the corpus spelling, with the type its config declares")
	testkit.Equal(t, pools[1], projection.PoolPlan{
		Role:  "payload",
		Field: "ValuePool",
		Members: [3]projection.Expr{
			`Value{Body: "test-body"}`, `Value{Body: "other-body"}`, `Value{Body: ""}`,
		},
		Type: golang.RefFor("string", ""),
	}, "the payload pool reaches inside the composite")
}

func TestPoolsShareOneStructAcrossMethods(t *testing.T) {
	t.Parallel()

	r, methods := poolFixture(t, poolField{"Key", "key", `"test-key"`})
	second := subject.Method{Sig: &golang.Sig{
		Name:   "Delete",
		Params: []golang.Param{{Name: "req", Source: storefixture.Named("PutRequest")}},
	}}
	pools, refusals := poolsOf(r, append(methods, second))
	testkit.Len(t, refusals, 0, "the shared struct refuses nothing twice either")
	testkit.Len(t, pools, 1, "two methods drawing one request share its pool")
}

func TestPoolsRefuseWhatTheyCannotSeed(t *testing.T) {
	t.Parallel()

	t.Run("a roled field with no default", func(t *testing.T) {
		t.Parallel()
		r, methods := poolFixture(t, poolField{"Key", "key", ""})
		pools, refusals := poolsOf(r, methods)
		testkit.Len(t, pools, 0, "pool[0] is the default verbatim, and there is none")
		testkit.Len(t, refusals, 1, "the gap is named")
		testkit.Contains(t, refusals[0].What, "PutRequest.Key", "down to the field")
		testkit.Contains(t, refusals[0].Remedy, "default", "and the remedy is the missing declaration")
	})

	t.Run("a default with no swap point", func(t *testing.T) {
		t.Parallel()
		r, methods := poolFixture(t, poolField{"Key", "key", `"localhost"`})
		pools, refusals := poolsOf(r, methods)
		testkit.Len(t, pools, 0, "two equal members fund no miss")
		testkit.Len(t, refusals, 1, "refused by name, not emitted broken")
	})
}
