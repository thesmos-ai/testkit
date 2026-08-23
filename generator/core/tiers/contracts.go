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

	// TypeArgIsValue says the type argument is what the store HOLDS rather
	// than what it is keyed by.
	//
	// A lease is keyed by what acquire takes and a cell by what its writer
	// sends, so for those the roles draw from the key pool. A publisher's
	// message is neither: it is the payload, and drawing it as a key
	// leaves the run with no value pool at all — which every law wanting
	// one then declines against, on a fixture whose oracle is right there.
	TypeArgIsValue bool

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

// ContractRoleOptional reports a role the oracle models where a
// declaration carries it and does without where it does not.
//
// [ContractRoles] reads the required set off the op table, which is right
// for a role the contract cannot be itself without: a lease with no
// release is not a lease. A redelivery is not like that — the mode laws
// declare it optional and bind in their unrefined form when it is absent
// — so listing its op would otherwise make three publishers that never
// mention it incomplete, and drop them to the twin floor.
func ContractRoleOptional(contract, role string) bool {
	return contractOptionalRoles[contract+"."+role]
}

//nolint:gochecknoglobals // a lookup table, read-only after init.
var contractOptionalRoles = map[string]bool{
	contractPublisher + "." + roleRedeliver: true,
}

// ContractRoleWrites reports a role whose call puts state in — the roles
// a run's history is made of, as opposed to the ones that observe it.
//
// A separate question from whether the role is spelled "writer", which is
// why a table rather than a suffix: a transaction's begin opens the write
// its terminal pair commits, and a pool's get takes a value out that its
// partner puts back.
//
// Two callers ask, and they ask for different reasons. The pool docblocks
// need to know whether the run writes at all, because the sentence about
// collision density describes a history an interface with no writer
// cannot have. The differential's defect rule needs somewhere to plant a
// dropped write, and additionally requires the contract to carry a store
// row — a reference that models nothing has nothing to still be holding.
//
// A publisher's publish is absent on purpose. It is driven by the
// delivery set rather than by an action of its own, and [writerCarrier]
// finds it through that.
func ContractRoleWrites(contract, role string) bool {
	return contractWritingRoles[contract+"."+role]
}

//nolint:gochecknoglobals // a lookup table, read-only after init.
var contractWritingRoles = map[string]bool{
	contractCAS + "." + roleWriter:      true,
	contractChain + "." + roleAppend:    true,
	contractLease + "." + roleAcquire:   true,
	contractUpserter + "." + roleWriter: true,
	contractUpdater + "." + roleWriter:  true,
	contractTx + "." + roleBegin:        true,
	contractPool + "." + roleGet:        true,
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
	rolePublish   = "publish"
	roleSubscribe = "subscribe"
	roleRedeliver = "redeliver"

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

	opGet       = "Get"
	opVerify    = "Verify"
	opPublish   = "Publish"
	opSubscribe = "Subscribe"
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
		// A publisher's oracle is the fan-out its declaration implies:
		// everyone subscribed at the moment of a publish gets that message,
		// once. Deliberately once — a subject claiming at-least-once may
		// deliver more and one claiming at-most-once may deliver less, and a
		// reference sitting at neither extreme is what lets the comparison
		// say which way a subject deviated.
		//
		// The message type comes from publish's argument. Subscribe's result
		// is a channel, which is a handle rather than a value, so a type
		// argument read off it would instantiate the store at something no
		// pool draws.
		contractPublisher: {
			Store:          "FanOut",
			TypeArgRole:    rolePublish,
			TypeArgIsValue: true,
			// No error arguments, and that is not an oversight the lenient
			// arm covers. A fan-out has no state to refuse from: a publish
			// reaches whoever is subscribed, and a subscribe always
			// succeeds. Where a publisher declares a lifecycle, the
			// sentinel it reports past Close is the lifecycle mixin's and
			// the laws that read it are its own.
		},
		contractCAS: {
			Store:       "VersionedCell",
			TypeArgRole: roleWriter,
			// A cell holds one value and is keyed by nothing, so the
			// writer's argument is the payload. Drawn as a key it reached
			// the right slot anyway, and every sentence the run wrote about
			// that pool then called a cas.Value a key.
			TypeArgIsValue: true,
			VersionParam:   "version",
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

		// Both forward unchanged: the oracle is written at the shapes a Go
		// publisher declares, so there is nothing for the adapter to
		// translate. See ref.FanOut, which says why it is shaped that way.
		contractPublisher + "." + rolePublish:   opPublish,
		contractPublisher + "." + roleSubscribe: opSubscribe,
		// A redelivery on a fan-out is another publish, and the oracle
		// says so rather than growing a second method that would do the
		// same thing. The duplicate is the point: at-least-once permits
		// it, exactly-once must swallow it, and the reference offering
		// the message twice is what either claim is measured against.
		contractPublisher + "." + roleRedeliver: opPublish,
	}
	contractRoleDrains = map[string]bool{
		// The AppendOnly oracle replays through an iterator; the corpus's
		// chain answers a slice, and the adapter drains the difference.
		contractChain + "." + roleReplay: true,
	}
)
