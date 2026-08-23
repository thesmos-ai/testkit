// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

import "slices"

// ContractStore returns the [engine/model/ref] store the named contract's
// roles delegate to, and whether one ships.
//
// A contract row exists only where the oracle is derivable whole: every role
// in the op table resolvable from the stamps, the store's type argument
// spoken by a role's own signature, and every constructor argument either a
// sentinel this generator can mint or a semantics choice it can make. The
// families that fail that bar — a pool needing a resource constructor, a
// saga needing its steps, a coalescer needing the function it coalesces —
// stay on the twin floor, whose header says so, and `ref=` raises them.
func ContractStore(contract string) (ContractStoreSpec, bool) {
	spec, shipped := contractStores[contract]
	return spec, shipped
}

// ContractStoreSpec is one derivable contract oracle.
type ContractStoreSpec struct {
	// Store is the ref type; "New" + Store its constructor — the naming
	// convention the shape oracles already rely on.
	Store string

	// TypeArgRole names the role whose signature speaks the store's one
	// type argument: its first parameter, or its first result when
	// TypeArgResult is set.
	TypeArgRole   string
	TypeArgResult bool

	// CtorFns are constructor arguments before the error slots, each the
	// name of a ref-package function instantiated at the store's type
	// argument and called with nothing — the chain's default hash, a
	// semantics choice the oracle owns rather than a fact the declaration
	// states.
	CtorFns []string

	// VersionParam names the contract parameter whose stamp names the
	// version field on the store's value type. When set, the constructor's
	// first argument is the generated projection of that field — the same
	// one-derivation rule the key projection follows.
	VersionParam string

	// Errs are the constructor's error arguments in declaration order. A
	// named entry mints a sentinel; an empty one renders nil — the oracle's
	// lenient arm, chosen where two legitimate dialects exist and the
	// stricter one would fail the weaker. The corpus proved the lease row:
	// releasing what was never held is ordinary Go to its subject, and a
	// strict oracle read the no-op as divergence.
	Errs []ContractErr

	// ShapeOps delegate the interface's non-role methods by pseudo-shape —
	// the cell's read is no role the cas contract declares, and an
	// aggregator-shaped method on a cell can only be asking the cell what it
	// holds. A shape absent here stays inert, with the header saying so.
	ShapeOps map[string]string

	// ConcModel names the linearize model the family's concurrent leg
	// checks against, empty where none derives. The generator wires the
	// leg only when the model's own op vocabulary resolves from the roles.
	ConcModel string
}

// ContractErr is one constructor error argument. NilUnder names a mixin
// whose presence on the Role method renders nil in the sentinel's place —
// the claim says the stricter dialect does not apply, and the oracle's nil
// arm is the lenient one. The corpus proved the row both ways: a strict
// lease refuses the second acquire, and an idempotent one re-enters.
type ContractErr struct {
	Suffix, Msg    string
	Role, NilUnder string

	// Param names the contract parameter whose stamp supplies the sentinel.
	// A declaration that stamps it gives the oracle and the law one error
	// identity; absent the stamp, the constructor mints its own.
	Param string
}

// ContractRoleOp returns the oracle method the named contract role delegates
// to.
func ContractRoleOp(contract, role string) (string, bool) {
	op, ok := contractRoleOps[contract+"."+role]
	return op, ok
}

// ContractRoleDrains reports that the role's oracle op streams through an
// iterator while the role method answers a slice, so the adapter collects
// rather than delegating the return directly.
func ContractRoleDrains(contract, role string) bool {
	return contractRoleDrains[contract+"."+role]
}

// ContractRoles returns the named contract's role vocabulary, sorted — the
// set an interface must resolve completely for the oracle to derive.
func ContractRoles(contract string) []string {
	prefix := contract + "."
	out := make([]string, 0, 2)
	for key := range contractRoleOps {
		if rest, matched := trimPrefix(key, prefix); matched {
			out = append(out, rest)
		}
	}
	slices.Sort(out)
	return out
}

// ContractsWithStores returns every contract carrying a store row, sorted,
// for the censuses.
func ContractsWithStores() []string {
	out := make([]string, 0, len(contractStores))
	for name := range contractStores {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

// trimPrefix is strings.CutPrefix without the import for two call sites.
func trimPrefix(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):], true
	}
	return "", false
}

// The contract-role vocabulary, spelled once: the role names are the
// directives' and the op names the oracles'.
const (
	roleAcquire = "acquire"
	opAcquire   = "Acquire"
	opRelease   = "Release"

	roleRelease = "release"
	roleGet     = "get"
	rolePut     = "put"
	roleAppend  = "append"
	roleReplay  = "replay"
	roleVerify  = "verify"
	roleWriter  = "writer"

	opGet    = "Get"
	opVerify = "Verify"
)

// The contract oracle tables.
//
//nolint:gochecknoglobals // lookup tables, read-only after init.
var (
	contractStores = map[string]ContractStoreSpec{
		contractLease: {
			Store:       "LeaseTracker",
			TypeArgRole: roleAcquire,
			ConcModel:   "LeaseTable",
			Errs: []ContractErr{
				{
					Suffix: "Held", Msg: "the model reference already holds the key",
					Role: roleAcquire, NilUnder: mixinIdempotent, Param: "held",
				},
				// Lenient release: giving up what was never taken is
				// ordinary Go to the corpus subject, and nil is the
				// tracker's spelling of that dialect.
				{},
			},
		},
		contractChain: {
			Store:       "AppendOnly",
			TypeArgRole: roleAppend,
			// The oracle's own bookkeeping hash: any deterministic chain
			// serves its Verify, and the default is the semantics choice.
			CtorFns: []string{"DefaultChainHash"},
		},
		contractCAS: {
			Store:        "VersionedCell",
			TypeArgRole:  roleWriter,
			VersionParam: "version",
			Errs: []ContractErr{
				{
					Suffix: "Mismatch", Msg: "the write's version is stale",
					Role: roleWriter, Param: "mismatch",
				},
				{Suffix: "Empty", Msg: "the cell holds nothing yet"},
			},
			// The cell's read is no role the contract declares; an
			// aggregator-shaped method on a cell can only be asking what it
			// holds.
			ShapeOps: map[string]string{shapeAggregator: opGet},
		},
	}
	contractRoleOps = map[string]string{
		contractLease + ".acquire": opAcquire,
		contractLease + ".release": opRelease,

		contractChain + "." + roleAppend: "Append",
		contractChain + "." + roleReplay: "Replay",
		contractChain + "." + roleVerify: opVerify,

		contractCAS + "." + roleWriter: fieldPut,
	}
	contractRoleDrains = map[string]bool{
		// The AppendOnly oracle replays through an iterator; the corpus's
		// chain answers a slice, and the adapter drains the difference.
		contractChain + "." + roleReplay: true,
	}
)
