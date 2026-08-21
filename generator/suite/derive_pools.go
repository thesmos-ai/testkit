// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"slices"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/stamp"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// DeriverPools attributes pool refusals. Not in the deriver registry:
// pools are an input projection the shell computes before any check
// derives, not a check family — but a gap in them is still a named
// gap, and it reports under this name.
const DeriverPools DeriverName = "pools"

// poolsOf projects the drawn pools from every drawn parameter: one
// [projection.PoolPlan] per roled declaration, its members derived by
// the projection's transforms — the default stamp verbatim, the
// distinctness swap, the hostile member.
//
// A parameter reaches a role two ways, and both are walked. A REQUEST
// STRUCT carries the role on the field holding the value, so the walk
// descends into it; two methods drawing one struct share its pools. A
// BARE PARAMETER has no field, so the role is declared on the named
// type it is written at — `type Key string` — which is the only place
// the parameter shape leaves. Reading only the first left every
// interface taking `(ctx, key Key)` with no pools at all, which is
// most of them.
//
// Refusals, never silence: a roled declaration with no default, a
// qualified default (a symbol, not a literal the transforms can
// splice), or a member a transform refuses each name the declaration
// and the consumer action that closes the gap.
func poolsOf(r golang.Resolver, methods []subject.Method) ([]projection.PoolPlan, []Refusal) {
	var pools []projection.PoolPlan
	var refusals []Refusal
	var seen []string

	keep := func(plan projection.PoolPlan, refusal *Refusal, roled bool) {
		switch {
		case !roled:
		case refusal != nil:
			refusals = append(refusals, *refusal)
		default:
			pools = append(pools, plan)
		}
	}

	// Parameters AND value returns. A role names what a value IS, not
	// which direction it travels, and the seed seam is the case that
	// proves it: an interface nothing can write to answers its payload
	// and never takes one, so walking arguments alone derives a key pool
	// with no values to pair — half a corpus, and a hit check that can
	// never be seeded.
	for _, m := range methods {
		sources := make([]*node.TypeRef, 0, len(m.Params)+len(m.Returns))
		for _, p := range m.CallArgs() {
			sources = append(sources, p.Source)
		}
		for _, v := range m.ValueReturns() {
			sources = append(sources, v.Source)
		}
		for _, src := range sources {
			decl, resolved := r.Resolve(src)
			if !resolved {
				continue
			}
			switch d := decl.(type) {
			case *sdk.Struct:
				if slices.Contains(seen, d.Name) {
					continue
				}
				seen = append(seen, d.Name)
				for _, f := range golang.ExportedFields(d) {
					keep(poolOf(d.Name, f.Name, f.Meta(), golang.RefFor(f.Type.Name, f.Type.Package)))
				}
			case *sdk.Alias:
				if slices.Contains(seen, d.Name) {
					continue
				}
				seen = append(seen, d.Name)
				// The named type itself is what the pool holds, not what
				// it is defined over: a `type Key string` pool is a
				// []Key, and a []string would not be assignable to the
				// parameter it is drawn for.
				keep(poolOf(d.Name, d.Name, d.Meta(), golang.RefFor(d.Name, d.Package)))
			}
		}
	}
	return pools, refusals
}

// poolOf derives one roled declaration's pool; roled reports false for
// a declaration this walk does not own.
//
// Takes the metadata bag rather than the node, because the two arms
// hand it a field and a named type and everything below reads the same
// two stamps off either.
func poolOf(
	owner, name string, bag *sdk.Bag, member sdk.Ref,
) (projection.PoolPlan, *Refusal, bool) {
	role := stamp.RoleOf(bag)
	if role == "" {
		return projection.PoolPlan{}, nil, false
	}
	// A type-level stamp names one declaration twice; saying "Key.Key"
	// would read as a field nobody wrote.
	what := owner + "." + name
	if owner == name {
		what = name
	}
	refuse := func(why, remedy string) (projection.PoolPlan, *Refusal, bool) {
		return projection.PoolPlan{}, &Refusal{
			Deriver: DeriverPools,
			What:    "the " + role + " pool from " + what,
			Why:     why,
			Remedy:  remedy,
		}, true
	}

	value := stamp.DefaultOf(bag)
	if value == "" {
		return refuse("the field has a role but no default value, and the first "+
			"value the checks use is that default",
			"add a //testkit:default beside the role")
	}
	if stamp.DefaultPackage(bag) != "" {
		return refuse("the default names a symbol from another package rather than "+
			"a value written out, so a second value cannot be derived from it",
			"write the default as a literal, or supply the values through the config")
	}
	distinct, ok := projection.DistinctMember(projection.Expr(value))
	if !ok {
		return refuse("no second value can be derived from this default that differs "+
			"from it, and two equal values would leave every not-found check finding something",
			"spell the default's textual payload test-*, or supply the pool through the config")
	}
	hostile, ok := projection.HostileMember(projection.Expr(value), role)
	if !ok {
		return refuse("no hostile member derives from the default's shape",
			"supply the pool through the config, hostile member included")
	}
	return projection.PoolPlan{
		Role:    role,
		Field:   projection.PoolFieldName(name),
		Members: [3]projection.Expr{projection.Expr(value), distinct, hostile},
		Type:    member,
	}, nil, true
}
