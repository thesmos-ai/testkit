// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"slices"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// Action is one method, driven and compared.
type Action struct {
	sdk.BaseEmit

	// KindName is `model.action.<shape>` — the template that renders the
	// constructor call, selected the way the harness selects its check
	// templates.
	KindName sdk.Kind

	// Method is the source identifier, and the name a failing step reports.
	Method string

	// Shape is the detector that chose the constructor, for the header.
	Shape string

	// Ctor is the `engine/model/action` constructor, as a qualified
	// expression.
	Ctor *sdk.Expr

	// Iface is the subject type, for the closure's parameter.
	Iface sdk.Ref

	// Key and Value are the closure's drawn-input and result types, nil where
	// the shape carries none; Value2 is the second result of the shapes that
	// answer two.
	Key, Value, Value2 sdk.Ref

	// Args is the per-position pool of a multi-argument writer: each drawn
	// from its own fixture pair, because one shared pool cannot serve several
	// types.
	Args []ActionArg

	// Pool names the shared pool the constructor draws from — "keys",
	// "values", or empty for a shape that draws nothing.
	Pool string

	// NoError marks a method answering without an error where the
	// constructor compares one: the closure supplies nil itself.
	NoError bool

	// TakesCtx reports whether the method's first parameter is a context. The
	// constructor's closure always receives one; a method that does not take
	// it ignores it.
	TakesCtx bool

	// Records marks the write a drain claim is judged against: the run's
	// log is handed to the constructor, which files every call both sides
	// accepted. Partitioned says that constructor also takes the
	// projection naming which partition to file it under.
	//
	// The log rides the constructor rather than a closure inside the
	// write, because the closure is handed the subject and then the
	// reference — a log filled from inside it holds every write twice,
	// which a membership check cannot see and a count reads as every
	// element having been dropped.
	Records     bool
	Partitioned bool

	// Sentinel is the declaration's stamped miss identity, armed on the
	// error-answering reader shapes: where both sides err, the pair must
	// also agree on whether the error is this one. Nil where nothing is
	// stamped, and the comparison stays presence-only.
	Sentinel *sdk.Expr

	// TxCommit and TxRollback spell the two-phase composite's terminal
	// methods, with their ctx flags beside them: the template threads one
	// begin's handle into its own drawn terminal, which is the driving a
	// standalone commit could never do — its handles came from a pool no
	// begin filled. Value carries the handle type.
	TxCommit, TxRollback       string
	TxCommitCtx, TxRollbackCtx bool

	// Partner is the sibling a two-role cycle returns to — the pool's put
	// beside its get — with PartnerCtx saying whether that call forwards
	// the run's context.
	//
	// The pair is one action because the claims about it are about the
	// CYCLE: a pool is balanced when every value taken comes back, and a
	// walk that draws a get and a put independently is never at rest
	// between steps, so a leak-free law checked there reports an
	// outstanding value as a leak on a correct subject.
	Partner    string
	PartnerCtx bool
}

// ActionPkg is the engine constructors' import path, for the option a
// template appends beside the closure.
func (*Action) ActionPkg() string { return actionPkg }

// ModelPkg surfaces the runner's import path to the action templates whose
// closures draw inline.
func (*Action) ModelPkg() string { return ModelPkg }

// Kind returns the shape-specific template key.
func (a *Action) Kind() sdk.Kind { return a.KindName }

// ActionArg is one drawn argument of a multi-argument writer or a
// parameterised pure call.
type ActionArg struct {
	// Field is the fixture accessor the position samples; Type its slice
	// literal's element clause. Wide blends the pair with arbitrary draws —
	// licensed for pure inputs unconditionally, because a pure call stores
	// nothing a claim could refuse.
	Field string
	Type  sdk.Ref
	Wide  bool
}

// OtherField is the accessor for the second, different value of the pair.
//
// Composed here rather than in the template, because the suffix belongs to
// the fixture's own naming policy and a template spelling it is a second
// home for a rule that has one.
func (a ActionArg) OtherField() string { return a.Field + subject.OtherSuffix }

// actionOf builds one method's action, or says why there is none.
func actionOf(ctx *sdk.GeneratorContext, b *Bindings, m *subject.Method) (*Action, string) {
	name := pseudoShape(m)
	if name == "" {
		return nil, "the annotator classified no shape for it"
	}
	if name == tiers.ShapeCollector {
		// The aggregator constructors compare a comparable result, and a
		// slice is not one; the stream action drains it instead.
		elem, err := collectorElem(b, m)
		if err != "" {
			return nil, err
		}
		return &Action{
			BaseEmit: b.BaseEmit,
			KindName: sdk.Kind(ActionKindPrefix + tiers.ShapeCollector),
			Method:   m.Name,
			Shape:    tiers.ShapeCollector,
			Ctor:     sdk.NewExternal(actionPkg, "Stream"),
			Iface:    b.IfaceRef,
			TakesCtx: m.TakesContext(),
			Value:    elem,
		}, ""
	}
	ctor, mapped := tiers.ActionFor(name)
	if !mapped {
		return nil, "no action drives the " + name + " shape"
	}
	for _, r := range m.Returns {
		if r.Source == nil || golang.IsError(r.Source) {
			// The error return is an interface too, and it is the one every
			// action already knows how to compare.
			continue
		}
		// A live handle — a channel, a function, an interface — compares by
		// identity, and two sides' handles never share one; the comparison
		// would fail every correct subject on its first answer.
		if golang.IsChannel(r.Source) || r.Source.IsFunc() || golang.IsInterface(r.Source) {
			return nil, "answers a live handle only identity could compare"
		}
	}

	a := &Action{
		BaseEmit: b.BaseEmit,
		KindName: sdk.Kind(ActionKindPrefix + name),
		Method:   m.Name,
		Shape:    name,
		Ctor:     sdk.NewExternal(actionPkg, ctor),
		Iface:    b.IfaceRef,
		TakesCtx: m.TakesContext(),
	}
	switch name {
	case shapeReader, "readernoerror", "pointerreader", "readerwithbool":
		a.Pool = poolKeys
		a.Key = m.CallArgs()[0].Type
		a.Value = m.Returns[0].Type
	case "multireader", "lookup":
		if len(m.Returns) != 3 {
			// The action drives a (value, value, error) triple and nothing
			// wider — a page-shaped read answers more, and its law walks it.
			return nil, "answers more than the (value, value, error) triple its action drives"
		}
		a.Pool = poolKeys
		a.Key = m.CallArgs()[0].Type
		a.Value = m.Returns[0].Type
		a.Value2 = m.Returns[1].Type
	case "batchreader":
		a.Pool = poolKeys
		a.Key = m.CallArgs()[0].Type
		elem, err := collectorElem(b, m)
		if err != "" {
			return nil, err
		}
		a.Value = elem
	case shapeWriter:
		a.Pool = poolValues
		a.Value = m.CallArgs()[0].Type
		// A writer whose mixin assigns it an oracle operation is one whose
		// argument is a key — a delete, not a put — and drawing values would
		// feed it strings no writer ever stored.
		for _, mixin := range m.Mixins {
			if _, assigned := tiers.KeyedStoreMixinOp(mixin); assigned {
				a.Pool = poolKeys
			}
		}
	case shapeAnsweringWriter:
		a.Pool = poolValues
		a.Value = m.CallArgs()[0].Type
	case "mutator":
		a.Pool = poolValues
		a.Value = m.CallArgs()[0].Type
	case shapeCompositeWriter:
		// Draws from both pools: the key beside the value is the shape.
		a.Pool = poolValues
		a.Key = m.CallArgs()[0].Type
		a.Value = m.CallArgs()[1].Type
	case "multiargwriter":
		// One shared pool cannot serve several types; each position draws
		// from its own fixture pair instead.
		for i, arg := range m.CallArgs() {
			a.Args = append(a.Args, ActionArg{Field: m.ArgFields[i], Type: arg.Type})
		}
	case shapeAggregator, "pure", "predicate":
		if m.HasResults() {
			a.Value = m.Returns[0].Type
		}
		a.NoError = name == shapeAggregator && len(m.Returns) == 1
		if name != shapeAggregator && len(m.CallArgs()) > 0 {
			// A parameterised pure call drives through the drawn-args
			// variant: each position draws from its own fixture pair,
			// blended with arbitrary values wherever the type can be seen
			// to the bottom — sound unconditionally, because a pure call
			// stores nothing a claim could refuse.
			for i, arg := range m.CallArgs() {
				a.Args = append(a.Args, ActionArg{
					Field: m.ArgFields[i],
					Type:  arg.Type,
					Wide:  unmakeable(ctx, shape.QName(arg.Source), map[string]bool{}) == "",
				})
			}
			ctor := "PureVar"
			if name == "predicate" {
				ctor = "PredicateVar"
			}
			a.KindName = sdk.Kind(ActionKindPrefix + name + "var")
			a.Ctor = sdk.NewExternal(actionPkg, ctor)
		}
	case "multiaggregator":
		a.Value = m.Returns[0].Type
		a.Value2 = m.Returns[1].Type
	case "streamreader":
		// The stream drains inside the closure, so the element type is the
		// stamp's — nothing else states what the iterator yields.
		q, stamped := b.valueQOf(m)
		if !stamped || q == "" {
			return nil, "streams elements no stamp names"
		}
		ref, err := golang.RefForQualified(q, b.IfaceName)
		if err != nil {
			return nil, "streams " + q + ", which no closure can spell: " + err.Error()
		}
		a.Value = ref
	case "streamconsumer":
		return nil, "consumes a caller-built stream no derivation can construct"
	case "lifecycle", "voidlifecycle", "poisonaccessor":
		// The call is the whole action.
	}
	return a, ""
}

// contractActionsOf re-points contract-role actions to the constructors that
// drive the role as itself. The actions were composed before the contract
// resolved its roles — the keyed-pool pass one loop up is the precedent —
// and the single-method rows are deliberately renames: the writer closure is
// already the constructor's shape, so only the name and the header change.
// The tx composite is the exception with teeth: the begin's action becomes
// the whole begin-terminal cycle and the terminal siblings' standalone
// actions are dropped, because a commit drawn from a value pool operates on
// a handle no begin minted — agreement over bogus handles was the entire
// content of that driving. A recording append keeps its recording closure:
// the rename touches the constructor, never the history log the writer
// template emits around it.
func contractActionsOf(b *Bindings, harness *subject.Projection) {
	for i := range harness.Methods {
		carrier := &harness.Methods[i]
		for _, contract := range carrier.Contracts {
			roles := contractRoleMethods(harness, carrier, contract)
			for role, rm := range roles {
				ctor, mapped := tiers.ContractActionFor(contract, role)
				if !mapped || rm == nil {
					continue
				}
				a := b.actionFor(rm.Name)
				if a == nil {
					continue
				}
				consumed := tiers.ContractActionConsumes(contract, role)
				switch len(consumed) {
				case 0:
					a.Ctor = sdk.NewExternal(actionPkg, ctor)
					a.Shape = contract + "." + role
					continue
				case 1:
					cycleActionOf(b, a, ctor, contract, role, roles[consumed[0]])
					continue
				}
				commit, rollback := roles[consumed[0]], roles[consumed[1]]
				if commit == nil || rollback == nil ||
					len(rm.Returns) == 0 || golang.IsError(rm.Returns[0].Source) {

					continue // half a trio, or a begin answering no handle
				}
				cAct, rAct := b.actionFor(commit.Name), b.actionFor(rollback.Name)
				if cAct == nil || rAct == nil {
					continue
				}
				a.Ctor = sdk.NewExternal(actionPkg, ctor)
				a.KindName = sdk.Kind(ActionKindPrefix + "twophase")
				a.Shape = contract + "." + role
				a.Pool = ""
				a.Value = rm.Returns[0].Type
				a.TxCommit, a.TxCommitCtx = commit.Name, commit.TakesContext()
				a.TxRollback, a.TxRollbackCtx = rollback.Name, rollback.TakesContext()
				reason := "driven through the " + rm.Name +
					" composite — a standalone terminal would operate on handles no begin minted"
				b.dropAction(commit.Name, reason)
				b.dropAction(rollback.Name, reason)
			}
		}
	}
}

// cycleActionOf folds a two-role pair into the one action that drives the
// cycle, and drops the partner's standalone action.
//
// The pool's get and put, today. Independently they are two writes the
// walk interleaves, and every claim the contract states is about the
// round trip: a value taken is outstanding until it comes back, so a pool
// driven by separate actions is never at rest and the leak-free law reads
// a legitimately-held value as a leak. Driven as a cycle, the pool is
// quiescent after every action, which is where the claim holds.
//
// A partner that answers no action, or a primary that yields nothing to
// hand back, leaves both standing: two ordinary actions state less than
// the cycle does, and nothing at all states nothing.
func cycleActionOf(
	b *Bindings, a *Action, ctor, contract, role string, partner *subject.Method,
) {
	if partner == nil || b.actionFor(partner.Name) == nil {
		return
	}
	a.Ctor = sdk.NewExternal(actionPkg, ctor)
	a.KindName = sdk.Kind(ActionKindPrefix + "cycle")
	a.Shape = contract + "." + role
	a.Pool = ""
	a.Partner, a.PartnerCtx = partner.Name, partner.TakesContext()
	b.dropAction(partner.Name, "driven through the "+a.Method+" cycle — a "+
		"standalone "+partner.Name+" would leave the pool holding values "+
		"no get took, and the balance claims are about the round trip")
}

// appendActionOf answers the driven offset-answering append of an appender
// contract, nil where the interface carries none. The offset type is held
// to int64 because the shared-history model counts in it; a log offsetting
// otherwise keeps its sequential law and no leg.
func appendActionOf(b *Bindings, harness *subject.Projection) (*Action, sdk.Ref) {
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if !slices.Contains(m.Contracts, "appender") {
			continue
		}
		if len(m.CallArgs()) != 1 || len(m.Returns) == 0 ||
			shape.QName(m.Returns[0].Source) != "int64" {

			continue
		}
		if a := b.actionFor(m.Name); a != nil && a.Pool != "" {
			return a, m.CallArgs()[0].Type
		}
	}
	return nil, nil
}

// pseudoShape is the detector's spelling, refined by the one fact the
// annotator does not state: an aggregator returning a slice is a collector,
// which drains rather than compares.
func pseudoShape(m *subject.Method) string {
	name := shape.Get(m.Source.Meta())
	if name == shapeAggregator && returnsSlice(m) {
		return tiers.ShapeCollector
	}
	return name
}

// collectorElem lifts the collector's element type into a renderable
// reference.
func collectorElem(b *Bindings, m *subject.Method) (sdk.Ref, string) {
	elem := shape.GoSliceElem(m.Returns[0].Source)
	ref, err := golang.RefForQualified(shape.QName(elem), b.IfaceName)
	if err != nil {
		return nil, "collects " + shape.QName(elem) + ", which no reference can spell: " + err.Error()
	}
	return ref, ""
}

// Skip is a method with no action, and the reason.
type Skip struct{ Method, Reason string }
