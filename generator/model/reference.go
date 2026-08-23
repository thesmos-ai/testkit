// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"slices"
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// Reference is how the run builds its oracle.
type Reference struct {
	// SuppliedCtor is the constructor the directive named, nil where the
	// oracle is derived. When set, nothing else here is: the arguments belong
	// to the consumer's own constructor.
	SuppliedCtor *sdk.Expr

	// MissSym is the declaration's own miss sentinel, routed into the
	// oracle's constructor where a mixin stamps one — the guard a
	// sentinel-checking law reads then matches the identity the fixture
	// declared, instead of a minted private error it can never equal.
	// Nil falls back to the minted MissName var.
	MissSym *sdk.Expr

	// TypeName, CtorName and MissName are the derived adapter's identifiers:
	// the struct over the oracle, its constructor, and the sentinel the oracle
	// reports for a key nothing wrote.
	TypeName, CtorName, MissName string

	// Oracle names which shipped store the adapter wraps; Dedupe and Pins are
	// its deduplicating and resolution-pinning refinements, each where a
	// mixin claims it. TwinWhy is the header's reason where the oracle is the
	// twin floor.
	Oracle  Oracle
	Dedupe  bool
	Pins    bool
	TwinWhy string

	// The contract oracle's own surface: ContractStore is the ref type,
	// ContractName the claim that selected it, ContractArg its one type
	// argument, and CtorErrs the constructor's error arguments in order —
	// a named entry is a minted sentinel, an unnamed one renders nil.
	ContractStore string
	ContractName  string
	ContractArg   sdk.Ref
	CtorErrs      []CtorErr

	// CtorFns are ref-package functions instantiated at ContractArg and
	// called with nothing, rendered before the error slots — the oracle's
	// own semantics choices, like the chain's default hash. VersionField is
	// the value field the version= stamp names; when set, the constructor's
	// first argument is the generated projection of it.
	CtorFns      []string
	VersionField string

	// KeyField is the field of the value type the map oracle keys on, empty
	// for the keyed oracle, whose key is an argument.
	KeyField string
}

// evictingReadOf names the method whose miss a bound makes legal: a keyed
// read answering `(V, bool)`, where absence is an answer rather than a
// failure. Empty where the interface offers none.
//
// The bool half is the whole condition. A read with an error channel
// cannot say "evicted" without saying which error that is, so the
// asymmetric comparison would have nothing to key on; a read answering a
// presence flag says it in a return the action already compares.
//
// Comparable, because [action.EvictingReader] constrains its value type
// where the symmetric reader does not — it compares hits by equality
// rather than deferring to a differ. A value no `==` accepts leaves the
// defeat standing, which is the twin and an honest header rather than a
// generated file that does not compile.
func evictingReadOf(harness *subject.Projection) string {
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if pseudoShape(m) != shapeReaderWithBool || len(m.Returns) == 0 {
			continue
		}
		if golang.IsComparable(m.Returns[0].Source) {
			return m.Name
		}
	}
	return ""
}

// Supplied reports that the directive named the reference.
func (r Reference) Supplied() bool { return r.SuppliedCtor != nil }

// IsContract reports the contract oracle: role-stamped delegation over a
// shipped store whose semantics are the claim's own.
func (r Reference) IsContract() bool { return r.Oracle == OracleContract }

// Twin reports the twin floor: the subject's own factory stands in.
func (r Reference) Twin() bool { return r.Oracle == OracleTwin }

// Derived reports that an adapter over a shipped oracle is generated — the
// case that owes a miss sentinel, an adapter and a companion proof.
func (r Reference) Derived() bool { return !r.Supplied() && !r.Twin() }

// StoreType is the wrapped oracle's type name, and "New" + StoreType its
// constructor — the naming convention `ref` keeps, relied on here so the
// template asks one question instead of branching twice.
func (r Reference) StoreType() string {
	switch r.Oracle {
	case OracleKeyed:
		return "KeyedStore"
	case OracleCollection:
		if r.Dedupe {
			return "SetCollection"
		}
		return "Collection"
	case OracleContract:
		return r.ContractStore
	case OracleMap, OracleTwin:
		// The twin has no store; nothing renders the answer.
	}
	if r.Pins {
		return "StickyStore"
	}
	return "MapStore"
}

// Keyed reports the keyed-put oracle, whose constructor takes no projection.
func (r Reference) Keyed() bool { return r.Oracle == OracleKeyed }

// Collects reports the append-and-drain oracle: one type argument, no keys,
// no miss sentinel — a drain of nothing is an empty slice, not an error.
func (r Reference) Collects() bool { return r.Oracle == OracleCollection }

// Oracle names a shipped reference implementation the adapter can wrap.
type Oracle string

// CtorErr is one error argument of a contract oracle's constructor. Name is
// the generated sentinel's identifier, empty where the slot renders nil.
type CtorErr struct {
	// Name is the minted sentinel's identifier, empty where the slot is nil
	// or the declaration stamped one. Sym is the stamped sentinel — the
	// declaration's own error, which gives the oracle and the bound law one
	// identity to agree on.
	Name, Msg string
	Sym       *sdk.Expr
}

// AdapterMethod is one interface method on the derived reference.
type AdapterMethod struct {
	// Sig is the source signature the body is composed from.
	Sig *golang.Sig

	// Op is the oracle method the body forwards to, empty for an inert body.
	// Collect marks an op that streams through an iterator while the method
	// answers a slice, so the body drains rather than returning the call.
	Op      string
	Collect bool

	// Folds marks a read whose miss the interface reports as a flag while
	// the oracle reports it as a sentinel, so the body converts rather than
	// returning the call.
	//
	// The seam is one line and it belongs here rather than at each
	// comparison: a caller of this interface cannot tell an evicted key from
	// one nothing wrote, and the oracle's sentinel is the same non-answer
	// spelled the way a store spells it.
	Folds bool

	// Reason says why an inert body is inert, for the comment above it.
	Reason string
}

// referenceOf fills the strongest reference the shape derives.
//
// Three oracles cover the store models Go interfaces actually declare. A
// composite writer — Put(ctx, k, v) — selects the keyed store, and needs no
// key projection: the key is an argument. A plain writer — Store(ctx, v) —
// selects the map, keyed on the one field of the value that holds the key.
// The composite wins where both appear, because a keyed oracle can host a
// value write only inertly while the reverse loses the delete. A shape no
// store models falls to the twin floor rather than refusing: a weaker
// differential the header names honestly beats an unarmed tier, and `ref=`
// raises the floor by hand.
func referenceOf(
	ctx *sdk.GeneratorContext,
	iface *sdk.Interface,
	harness *subject.Projection,
	b *Bindings,
	keyed, valued, composite, collector *subject.Method,
	partners map[string]string,
) bool {
	if named, given := directiveValue(iface, RefKey); given {
		if strings.Contains(named, ".") {
			ctx.Diag.Errorf(iface.Pos(),
				"%s: %s=%q on %q carries a qualifier; name a constructor in the "+
					"interface's own package",
				Name, RefKey, named, iface.Name)
			return false
		}
		b.Reference = Reference{SuppliedCtor: sdk.NewExternal(iface.Package, named)}
		return true
	}

	lower := strings.ToLower(harness.IfaceName[:1]) + harness.IfaceName[1:]
	names := Reference{
		TypeName: lower + "ModelReference",
		// Exported: the generated companion lands in the external test
		// package and proves the oracle from there, the way a consumer
		// would — and a consumer comparing against it gets the same door.
		CtorName: "New" + harness.IfaceName + "ModelReference",
		MissName: lower + "ModelMiss",
		MissSym:  missSentinelOf(harness),
	}

	twin := func(why string) bool {
		b.Reference = Reference{Oracle: OracleTwin, TwinWhy: why}
		return true
	}

	// A version-stamped fixture is a claim no store oracle survives: the
	// subject assigns the ordering member on write, a value-storing oracle
	// holds the input's zero, and the first read-back diverges on a correct
	// store. Twins stamp together. The key member is derived here, where
	// the reader is in hand — the classifier needs the same identity the
	// upsert inference reads, and the twin carries no adapter to ask. This
	// arm runs before the defeat scan because a session mixin can sit in
	// both tables, and only this arm derives what the classifier needs.
	if vm, member, stamped := sessionVersionOf(harness); stamped {
		if q, _ := b.valueQOf(vm); q != "" && !versionFieldDiag(ctx, iface, q, member) {
			return false
		}
		if keyed != nil {
			if q, _ := b.valueQOf(keyed); q != "" {
				b.sessionKeyField, _ = upsertKeyField(ctx, b, q)
			}
		}
		return twin("the subject assigns the version member on write, which no value-storing oracle stamps")
	}

	// A claim that defeats store modeling outranks every remaining
	// derivation: the twins lag together, where an immediate oracle reads
	// the claim's own slack as divergence.
	//
	// Unless the interface offers a read whose miss is legal, which lifts
	// the one defeat that is a fact about the read shape rather than about
	// what the subject does — see [tiers.OracleDefeat].
	evicting := evictingReadOf(harness)
	for i := range harness.Methods {
		for _, mixin := range harness.Methods[i].Mixins {
			defeat, defeated := tiers.DefeatsOracles(mixin)
			switch {
			case !defeated:
			case defeat.LiftedByEvictingRead && evicting != "":
				b.EvictingRead = evicting
			default:
				// A second claim defeats it outright, so the lift the first
				// earned is withdrawn with it: the twin compares everything
				// symmetrically, and a one-sided read beside it would be an
				// asymmetry against the subject's own factory.
				b.EvictingRead = ""
				return twin(defeat.Why)
			}
		}
	}

	// A contract claim outranks the shapes: its roles say what each method
	// is FOR, and the shipped store carries the claim's own semantics —
	// which is more than any shape-derived map can promise.
	handled, lenified, refused := contractOf(ctx, iface, b, harness, partners, names)
	if refused {
		return false
	}
	if handled {
		return true
	}
	if lenified != "" {
		return twin(lenified)
	}

	if (keyed == nil || historyDrained(harness)) && collector != nil && valued != nil {
		// A value writer beside a collector. Ordinarily nothing keyed reads
		// beside them — and where a history claim stands, a keyed read does
		// not change the election: the claim says the drain is an event log,
		// a map oracle collapses the log's repeats, and the corpus caught
		// exactly that when the isolation fixture grew its read. The one
		// agreement to check is that the writer adds what the collector
		// returns; the keyed read stays inert on the log oracle, and the
		// header says so.
		wroteV, _ := b.valueQOf(valued)
		elem := shape.QName(shape.GoSliceElem(collector.Returns[0].Source))
		if wroteV == "" || wroteV != elem {
			return twin("the drain returns " + elem + " where the writer adds " + wroteV)
		}
		// A derivable key field means upsert semantics — a second add under
		// a held key replaces — and the map is the store that models it. A
		// value with no key is a log entry, deduplicated only where a claim
		// licenses the collapse. The claims scan interface-wide, the way the
		// negation table does: a refinement rides whichever method carries
		// the stamp, and holds over the whole store. The corpus taught both
		// forks — every keyed-map subject diverged from a log at the first
		// repeated add, and the first history subject held two identical
		// events the inferred upsert map collapsed to one.
		history, dedupe := historyDrained(harness), false
		for i := range harness.Methods {
			m := &harness.Methods[i]
			for _, c := range append(append([]string{}, m.Mixins...), m.Contracts...) {
				dedupe = dedupe || tiers.CollectionDedupes(c)
			}
		}
		if !history {
			if field, keyRef := upsertKeyField(ctx, b, wroteV); field != "" {
				names.Oracle = OracleMap
				names.KeyField = field
				b.Keys.Type = keyRef
				b.Reference = names
				b.Adapter = adapterOf(b, harness, partners, OracleMap, wroteV)
				return true
			}
		}
		names.Oracle = OracleCollection
		names.Dedupe = dedupe
		b.Reference = names
		b.Adapter = adapterOf(b, harness, partners, OracleCollection, wroteV)
		return true
	}

	if keyed != nil && composite != nil {
		keyQ, _ := b.keyQOf(keyed)
		readV, _ := b.valueQOf(keyed)
		putK, _ := b.keyQOf(composite)
		putV, _ := b.valueQOf(composite)
		if keyQ != putK || readV == "" || readV != putV {
			return twin("the reader speaks (" + keyQ + " → " + readV +
				") where the keyed writer takes (" + putK + ", " + putV + ")")
		}
		names.Oracle = OracleKeyed
		b.Reference = names
		b.Adapter = adapterOf(b, harness, partners, OracleKeyed, putV)
		return true
	}

	if keyed == nil || valued == nil {
		return twin("no reader/writer pair derives a store")
	}

	keyQ, _ := b.keyQOf(keyed)
	readV, _ := b.valueQOf(keyed)
	wroteV, _ := b.valueQOf(valued)
	if readV == "" || readV != wroteV {
		return twin("the reader answers " + readV + " where the writer takes " + wroteV)
	}

	field, why := keyFieldOf(ctx, readV, keyQ)
	if field == "" {
		return twin("the key projection is underivable — " + why)
	}

	names.Oracle = OracleMap
	names.KeyField = field
	for _, mixin := range keyed.Mixins {
		// The reader's claim refines the oracle the way the drain's dedupe
		// claim refines the collection: sticky resolution is a different
		// store, not a different pool.
		if tiers.MapStorePins(mixin) {
			names.Pins = true
		}
	}
	b.Reference = names
	b.Adapter = adapterOf(b, harness, partners, OracleMap, readV)
	return true
}

// contractOf derives the contract oracle where an interface's stamps resolve
// a shipped store's whole role vocabulary: the carrier's role= names its own
// part, the partner keys name the siblings, and every role must land on a
// method or the family stays underived — half a lease checks nothing a twin
// does not. The store's one type argument is spoken by the type-arg role's
// own signature, and the constructor's error arguments are minted sentinels
// or the lenient nil, per the family's row. refused reports a directive
// invalid by name — a version= that no field projection can satisfy — and
// aborts the whole binding rather than falling through to a weaker oracle.
func contractOf(
	ctx *sdk.GeneratorContext,
	iface *sdk.Interface,
	b *Bindings,
	harness *subject.Projection,
	partners map[string]string,
	names Reference,
) (handled bool, lenified string, refused bool) {
	for i := range harness.Methods {
		carrier := &harness.Methods[i]
		for _, contract := range carrier.Contracts {
			spec, shipped := tiers.ContractStore(contract)
			if !shipped {
				continue
			}
			roles := contractRoleMethods(harness, carrier, contract)
			complete := true
			for _, role := range tiers.ContractRoles(contract) {
				complete = complete && roles[role] != nil
			}
			src := roles[spec.TypeArgRole]
			if !complete || src == nil {
				continue
			}
			var arg sdk.Ref
			var argQ string
			switch {
			case spec.TypeArgResult && len(src.Returns) > 0:
				arg = src.Returns[0].Type
				argQ = shape.QName(src.Returns[0].Source)
			case !spec.TypeArgResult && len(src.CallArgs()) > 0:
				arg = src.CallArgs()[0].Type
				argQ = shape.QName(src.CallArgs()[0].Source)
			}
			if arg == nil {
				continue
			}

			if spec.VersionParam != "" {
				field, stamped := stampValue(harness, carrier,
					shape.ContractParamKey(contract, spec.VersionParam).Name())
				if !stamped {
					// The cell cannot guard what nothing names; the twins
					// stand in and the header's floor says so.
					continue
				}
				if !versionFieldDiag(ctx, iface, b.substQ(argQ), field) {
					return false, "", true
				}
				names.VersionField = field
			}
			names.CtorFns = spec.CtorFns
			names.Oracle = OracleContract
			names.ContractStore = spec.Store
			names.ContractName = contract
			names.ContractArg = arg
			if !spec.TypeArgResult {
				// The store's type argument is a role argument, so the roles
				// draw keys: record the source for the pool derivation and
				// the methods whose actions draw from it.
				b.contractKeySrc = src
				b.contractKeyedRoles = map[string]bool{}
				for _, rm := range roles {
					b.contractKeyedRoles[rm.Name] = true
				}
			}

			lower := strings.ToLower(b.IfaceName[:1]) + b.IfaceName[1:]
			minted := false
			for _, e := range spec.Errs {
				ce := CtorErr{Msg: e.Msg}
				if e.Suffix != "" && !roleClaims(roles[e.Role], e.NilUnder) {
					// The declaration's own sentinel where one is stamped —
					// the bound law compares identities, and an oracle
					// disagreeing under a different spelling of the same
					// state would fail every correct subject.
					if sym, stamped := stampedSentinel(harness, carrier, contract, e.Param); stamped {
						ce.Sym = sym
					} else {
						ce.Name = lower + "Model" + e.Suffix
					}
					minted = true
				}
				names.CtorErrs = append(names.CtorErrs, ce)
			}
			if len(spec.Errs) > 0 && !minted {
				// Every sentinel lenified away is an oracle that can never
				// disagree — the kill matrix proved a fully-nil tracker
				// cannot see its own methods go inert. The twins say so
				// instead of a store pretending to check.
				return false, "the claims lenify every sentinel the " + contract +
					" oracle could disagree with", false
			}
			b.Reference = names
			b.Adapter = contractAdapterOf(harness, partners, contract, roles)
			// The concurrent leg's roles, recorded only for an oracle that
			// held: a lenified family fell to the twins above, and a leg
			// wired against a sentinel nothing minted renders nothing valid.
			if spec.ConcModel == "LeaseTable" && roles[roleLeaseAcquire] != nil &&
				roles[roleLeaseRelease] != nil {

				b.concAcquireName = roles[roleLeaseAcquire].Name
				b.concReleaseName = roles[roleLeaseRelease].Name
			}
			return true, "", false
		}
	}
	return false, "", false
}

// stampedSentinel resolves a contract error's declared sentinel, false where
// the parameter is unnamed or unstamped.
func stampedSentinel(
	harness *subject.Projection, carrier *subject.Method, contract, param string,
) (*sdk.Expr, bool) {
	if param == "" {
		return nil, false
	}
	v, ok := stampValue(harness, carrier, shape.ContractParamKey(contract, param).Name())
	if !ok {
		return nil, false
	}
	pkg, name, qualified := splitQualified(v)
	if !qualified {
		return nil, false
	}
	return sdk.NewExternal(pkg, name), true
}

// contractAdapterOf builds the role-stamped delegation table: a method
// filling a role forwards to the role's op, a non-role method whose shape
// the spec's ShapeOps claim forwards likewise — the cell's read is no role
// the cas contract declares — and everything else is inert.
func contractAdapterOf(
	harness *subject.Projection,
	partners map[string]string,
	contract string,
	roles map[string]*subject.Method,
) []AdapterMethod {
	spec, _ := tiers.ContractStore(contract)
	opOf := map[string]string{}
	drains := map[string]bool{}
	for role, m := range roles {
		if op, ok := tiers.ContractRoleOp(contract, role); ok {
			opOf[m.Name] = op
			drains[m.Name] = tiers.ContractRoleDrains(contract, role)
		}
	}
	out := make([]AdapterMethod, 0, len(harness.Methods))
	for i := range harness.Methods {
		m := &harness.Methods[i]
		am := AdapterMethod{Sig: m.Sig}
		op := opOf[m.Name]
		if op == "" {
			op = spec.ShapeOps[pseudoShape(m)]
		}
		switch role, partner := partners[m.Name]; {
		case partner:
			am.Reason = role
		case !m.TakesContext():
			am.Reason = "it takes no context to forward to the oracle"
		case op == "":
			am.Reason = "the " + contract + " oracle models only its roles"
		default:
			am.Op = op
			am.Collect = drains[m.Name]
		}
		out = append(out, am)
	}
	return out
}

// adapterOf builds the delegation table: every method forwards to the oracle's
// matching operation or holds an inert body. valueQ is the value spelling the
// oracle models — a writer of any other type stays inert, because forwarding
// it would hand the store a value its element clause refuses to compile.
func adapterOf(
	b *Bindings, harness *subject.Projection, partners map[string]string, oracle Oracle, valueQ string,
) []AdapterMethod {
	out := make([]AdapterMethod, 0, len(harness.Methods))
	for i := range harness.Methods {
		m := &harness.Methods[i]
		am := AdapterMethod{Sig: m.Sig}
		op, fromMixin := oracleOp(oracle, m)
		wroteQ, _ := b.valueQOf(m)
		switch role, partner := partners[m.Name]; {
		case partner:
			am.Reason = role
		case !m.TakesContext():
			am.Reason = "it takes no context to forward to the oracle"
		case op == "":
			am.Reason = "the oracle does not model its shape"
		case !fromMixin && pseudoShape(m) == shapeWriter && wroteQ != valueQ:
			am.Reason = "takes " + wroteQ + " where the oracle holds " + valueQ
		default:
			am.Op = op
			am.Folds = pseudoShape(m) == shapeReaderWithBool
		}
		out = append(out, am)
	}
	return out
}

// oracleOp resolves one method's delegation for the chosen oracle: a mixin
// assignment first — the stamp says what a method is for, outranking what it
// looks like — then the oracle's shape table. The second result reports the
// mixin route, whose argument is a key no value check applies to.
func oracleOp(oracle Oracle, m *subject.Method) (string, bool) {
	if oracle == OracleKeyed {
		for _, name := range m.Mixins {
			if op, assigned := tiers.KeyedStoreMixinOp(name); assigned {
				return op, true
			}
		}
	}
	var op string
	switch oracle {
	case OracleKeyed:
		op, _ = tiers.KeyedStoreOp(pseudoShape(m))
	case OracleCollection:
		op, _ = tiers.CollectionOp(pseudoShape(m))
	case OracleMap:
		op, _ = tiers.MapStoreOp(pseudoShape(m))
	case OracleContract, OracleTwin:
		// The contract adapter resolves by role, not by shape, and the twin
		// has no adapter at all; neither reaches this table.
	}
	return op, false
}

// upsertKeyField finds the identity field of an unread value type — the
// conventional ID or Key spelling — and lifts its type, for the writer-plus-
// drain interfaces whose subjects upsert by it. No reader states the key
// type, so the convention is the whole signal; a value keyed otherwise falls
// to the collection oracle, and a log subject with an incidental Key field
// falls to ref= — the header names the store either way.
func upsertKeyField(ctx *sdk.GeneratorContext, b *Bindings, valueQ string) (string, sdk.Ref) {
	for cand := range ctx.Reader.Structs().All() {
		if cand.Package+"."+cand.Name != valueQ {
			continue
		}
		for _, preferred := range keyFieldConventions {
			for _, f := range cand.Fields {
				if f.Name != preferred {
					continue
				}
				ref, err := golang.RefForQualified(shape.QName(f.Type), b.IfaceName)
				if err != nil {
					return "", nil
				}
				return f.Name, ref
			}
		}
	}
	return "", nil
}

// keyFieldOf finds the one field of the value struct that can hold the key,
// returning the failure's spelling when it cannot.
//
// One candidate answers directly. Several prefer the conventional spellings —
// ID, then Key — because a value type keyed on one of two same-typed fields is
// keyed on the one its author named for the job; a value keyed otherwise is
// what [RefKey] exists for.
func keyFieldOf(ctx *sdk.GeneratorContext, valueQ, keyQ string) (string, string) {
	var s *sdk.Struct
	for cand := range ctx.Reader.Structs().All() {
		if cand.Package+"."+cand.Name == valueQ {
			s = cand
			break
		}
	}
	if s == nil {
		return "", "no struct declaration was found for it"
	}

	var candidates []string
	for _, f := range s.Fields {
		if shape.QName(f.Type) == keyQ {
			candidates = append(candidates, f.Name)
		}
	}
	switch len(candidates) {
	case 0:
		return "", "no field of it has the key's type"
	case 1:
		return candidates[0], ""
	}
	for _, preferred := range keyFieldConventions {
		for _, cand := range candidates {
			if cand == preferred {
				return cand, ""
			}
		}
	}
	return "", "several fields share the key's type and none is named ID or Key"
}

// contractRoleMethods resolves the named contract's roles to methods: every
// method filling a role by its own stamp, plus each partner key the carrier
// names. Both walks, because a protocol splits its directives — the chain
// stamps append on one method and verify on another, and reading only the
// carrier's would leave a role the interface plainly fills unresolved.
func contractRoleMethods(
	harness *subject.Projection, carrier *subject.Method, contract string,
) map[string]*subject.Method {
	out := map[string]*subject.Method{}
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if role, ok := shape.ContractRoleKey(contract).Get(m.Source.Meta()); ok && role != "" {
			out[role] = m
		}
	}
	for _, role := range tiers.ContractRoles(contract) {
		v, ok := shape.ContractPartnerKey(contract, role).Get(carrier.Source.Meta())
		if !ok || v == "" {
			continue
		}
		if m := methodOf(harness, golang.LocalName(v)); m != nil {
			out[role] = m
		}
	}
	return out
}

// roleClaims reports whether the role's method carries the named mixin — the
// stamp that flips a constructor sentinel to the oracle's lenient nil.
func roleClaims(m *subject.Method, mixin string) bool {
	return m != nil && mixin != "" && slices.Contains(m.Mixins, mixin)
}
