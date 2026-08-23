// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"strings"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// lawsOf selects and fills every law the interface's classifications earn.
//
// Selection is [tiers.Select] over each non-partner method's whole
// classification set. A selected rule that cannot be filled lands in
// [Bindings.Unbound] with what it is waiting on — rendered in the header,
// because a law that quietly failed to bind reads as a claim the run checks.
func lawsOf(b *Bindings, harness *subject.Projection, partners map[string]string, keyed *subject.Method) {
	// Selection composes per method, but a claim holds over the interface —
	// the sticky stamp rides the reader and negates the writer-earned
	// observability law — so the conflict scan runs against every method's
	// mixins, partners included: an excluded method's claim still holds.
	claims := map[string]bool{}
	for i := range harness.Methods {
		for _, name := range harness.Methods[i].Mixins {
			claims[name] = true
		}
	}
	// The derived adapter's inert arms: a law reaching one compares against
	// a body answering zeros — and the companion drives the adapter as a
	// full subject, where the lie becomes a red run on the reference itself.
	inert := map[string]string{}
	if b.Reference.Derived() {
		for _, am := range b.Adapter {
			if am.Op == "" {
				inert[am.Sig.Name] = am.Reason
			}
		}
	}
	// One outcome per (law, selecting method): a contract classification
	// rides every role method, and re-selecting the same rule from each
	// would register one law twice and print one refusal per carrier.
	seen := map[string]bool{}
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if _, partner := partners[m.Name]; partner && len(m.Mixins) == 0 {
			// A role-overridden partner — a validator, a tamper — carries no
			// claim of its own and selects nothing. A partner that hosts its
			// own mixin still voices it: the leakfree open half names itself,
			// and excluding it from the sequences must not silence its law.
			continue
		}
		selectable := m.Classifications()
		if _, partner := partners[m.Name]; partner {
			selectable = m.Mixins
		}
		for _, r := range tiers.Select(selectable, subject.LawParams(harness.Methods, *m)) {
			if reason, negated := negatedBy(claims, r.Law); negated {
				if !seen[r.Law+"\x00"+reason] {
					seen[r.Law+"\x00"+reason] = true
					b.Unbound = append(b.Unbound, Skip{Method: r.Law, Reason: reason})
				}
				continue
			}
			before := len(b.Unbound)
			binding, ok := lawOf(b, harness, r, m, keyed, inert)
			if !ok {
				// lawOf appended the refusal; keep it only if new.
				added := b.Unbound[before:]
				b.Unbound = b.Unbound[:before]
				for _, u := range added {
					if !seen[u.Method+"\x00"+u.Reason] {
						seen[u.Method+"\x00"+u.Reason] = true
						b.Unbound = append(b.Unbound, u)
					}
				}
				continue
			}
			key := r.Law + "\x00bound\x00" + bindingFingerprint(binding)
			if seen[key] {
				continue
			}
			seen[key] = true
			// An optional role resolves only from the directive's own
			// carrier, and the same rule re-selected from a partner binds
			// the law again without it — one law twice, disagreeing about
			// its refinement, and the poorer twin's header calling armed
			// work unarmed. The richer binding subsumes the poorer in
			// either arrival order; genuinely distinct same-ID bindings —
			// one per method — share no field spelling and both stay.
			subsumed := false
			for i, held := range b.Laws {
				if lawSubsumes(held, binding) {
					subsumed = true
					break
				}
				if lawSubsumes(binding, held) {
					b.Laws[i] = binding
					subsumed = true
					break
				}
			}
			if subsumed {
				continue
			}
			b.Laws = append(b.Laws, binding)
			for _, field := range binding.Fields {
				// The flags ride the appended binding, never the attempt: a
				// handle filled for a law that later refused would render
				// locals nothing uses.
				if field.Kind() == sdk.Kind(LawFieldKindPrefix+"Compute") {
					b.Coalesced = true
				}
				if field.Kind() == sdk.Kind(LawFieldKindPrefix+"HistoryRef") {
					b.RecordsHistory = true
					b.HistoryElem = field.Value
					for _, a := range b.Actions {
						if a.Method != field.Method {
							continue
						}
						a.Records = true
						// The recording constructor, not the plain one with a
						// logging closure: the closure is handed the subject
						// and then the reference, so a log filled from inside
						// it holds every write twice. Both variants take the
						// log beside the pool; only the chain's also takes the
						// projection saying which partition a write lands in.
						if ctor, has := tiers.RecordingActionFor(a.Shape); has {
							a.Ctor = sdk.NewExternal(actionPkg, ctor)
							a.Partitioned = a.Shape != shapeWriter
						}
					}
				}
			}
		}
	}
	// A contract's classification rides every role method, and only the
	// directive host carries the stamps its roles resolve through — so a law
	// bound from the host still records a refusal per partner carrier. One
	// binding is the interface's outcome; the carrier refusals are noise.
	bound := map[string]bool{}
	for _, lb := range b.Laws {
		bound[lb.ID] = true
	}
	kept := b.Unbound[:0]
	for _, u := range b.Unbound {
		if !bound[u.Method] {
			kept = append(kept, u)
		}
	}
	b.Unbound = kept
}

// lawOf fills one rule, false where [Bindings.Unbound] records why not.
func lawOf(
	b *Bindings,
	harness *subject.Projection,
	r tiers.Rule,
	m, keyed *subject.Method,
	inert map[string]string,
) (*LawBinding, bool) {
	spec, specified := tiers.BindingFor(r.Law)
	if !specified {
		b.Unbound = append(b.Unbound, Skip{
			Method: r.Law,
			Reason: "the catalogue carries no instantiation spec for it",
		})
		return nil, false
	}

	pkg := LawPkg
	if spec.Timeaware {
		pkg = TimeawarePkg
	}
	lb := &LawBinding{
		BaseEmit: b.BaseEmit,
		ID:       r.Law,
		Ctor:     sdk.NewExternal(pkg, spec.Type),
		Ptr:      spec.Ptr,
		// The subject leads every law's argument list; the spec spells only
		// what follows it.
		Args: []sdk.Ref{b.IfaceRef},
	}
	if m != nil {
		// The selecting method, kept for the wording its row reports
		// under: the claim names what only this stamp supplies — which
		// method Close is, what the produced handle is called.
		lb.carrier = *m
	}
	for _, a := range spec.Args {
		ref, reason := resolveArg(b, harness, r, a, m, keyed)
		if reason != "" {
			b.Unbound = append(b.Unbound, Skip{Method: r.Law, Reason: reason})
			return nil, false
		}
		lb.Args = append(lb.Args, ref)
	}

	// Fields the law drives on the subject alone. An oracle that cannot model
	// one of these cannot fall behind, because the call's whole effect is one
	// the subject discards before Check returns.
	sutOnly := map[string]bool{}
	for _, f := range r.Fields {
		if f.SUTOnly {
			sutOnly[f.Name] = true
		}
		field, reason := lawFieldOf(b, harness, r, f, m, keyed)
		if reason != "" {
			b.Unbound = append(b.Unbound, Skip{Method: r.Law, Reason: reason})
			return nil, false
		}
		if field != nil {
			lb.Fields = append(lb.Fields, field)
		} else if f.Optional && f.Kind == tiers.KindRole {
			lb.Unarmed = append(lb.Unarmed, f.Name)
		}
	}
	for _, field := range lb.Fields {
		if field.Kind() == sdk.Kind(LawFieldKindPrefix+"Advance") {
			// A clock-bound law arms only where the ModelClocked option
			// supplies a subject on the run's clock; the template guards it.
			lb.Clocked = true
			b.UsesClock = true
		}
		if field.Kind() == sdk.Kind(LawFieldKindPrefix+"Classify") {
			lb.Session = true
		}
		if field.Kind() == sdk.Kind(LawFieldKindPrefix+"SuppliedField") {
			lb.Supplied = append(lb.Supplied, field.Pool)
		}
		if sutOnly[field.Name] {
			continue
		}
		if reason, held := inert[field.Method]; field.Method != "" && held {
			b.Unbound = append(b.Unbound, Skip{
				Method: r.Law,
				Reason: field.Name + " reaches " + field.Method +
					", which the derived reference answers inertly — " + reason,
			})
			return nil, false
		}
	}
	return lb, true
}

// bindingFingerprint spells what makes two bindings the same law twice: the
// methods its fields close over. Two writers earning one law separately are
// two bindings; one contract riding two roles is one.
func bindingFingerprint(lb *LawBinding) string {
	var out strings.Builder
	for _, f := range lb.Fields {
		out.WriteString(f.Name + "=" + f.Method + ";")
	}
	return out.String()
}

// lawSubsumes reports whether have covers want: the same law, with every
// resolved field of want — name and method both — present in have. A field
// have carries beyond that is the optional refinement want's carrier could
// not resolve.
func lawSubsumes(have, want *LawBinding) bool {
	if have.ID != want.ID || len(want.Fields) > len(have.Fields) {
		return false
	}
	got := make(map[string]bool, len(have.Fields))
	for _, f := range have.Fields {
		got[f.Name+"\x00"+f.Method] = true
	}
	for _, f := range want.Fields {
		if !got[f.Name+"\x00"+f.Method] {
			return false
		}
	}
	return true
}

// negatedBy resolves the first conflict row a held claim triggers, in the
// table's own order so the generated header is deterministic.
func negatedBy(claims map[string]bool, law string) (string, bool) {
	for _, n := range tiers.LawNegations() {
		if n.Law == law && claims[n.Mixin] {
			return n.Reason, true
		}
	}
	return "", false
}
