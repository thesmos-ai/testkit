// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/lifecycleafterclose"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
	"go.thesmos.sh/testkit/generator/suite"
)

// The planted defects this tier's rows are proven by.
//
// A check that always passes is indistinguishable from a working one
// until something breaks in production, and a run reporting "12 legs, 12
// passed" does not tell the two apart. Each rule below is the smallest
// implementation that breaks exactly one claim — the generated double
// with a single method overridden — and the row it belongs to ships
// Proven only because the rule reached it.
//
// Mechanical, from shape and stamps alone. The residue ships Argued and
// says so: a domain composite no rule reaches is an honest gap, and a
// proof nobody derived worn as a stamp is not.

// The stamps a rule reads, from the plugins that own them.
const (
	mixinAfterClose         = lifecycleafterclose.Name
	mixinAfterCloseSentinel = lifecycleafterclose.ParamSentinel
)

// defectFor is the planted defect for one law's row, and the method the
// defect overrides — nil where no rule reaches the law, or where the
// rule's target is not on this interface.
//
// The method travels with the defect because a defect is planted THROUGH
// one: the double's option is per-method, and a rule that picked its
// target then handed back only the plan would leave the caller to guess
// which method it meant.
func defectFor(b *Bindings, l *LawBinding) (projection.Defect, *subject.Method, bool) {
	rule, ruled := lawDefects()[l.ID]
	if !ruled {
		return nil, nil, false
	}
	return rule(b, l)
}

// lawDefectRule plants one law's defect from the interface's own stamps.
type lawDefectRule func(b *Bindings, l *LawBinding) (projection.Defect, *subject.Method, bool)

// lawDefects is the rule table, keyed by law. A law without a row is the
// honest residue — the domain composites, like the cursor's hand-built
// handle types and the lease's lying accounting, that no mechanical rule
// reaches.
func lawDefects() map[string]lawDefectRule {
	return map[string]lawDefectRule{
		lawid.LifecycleAfterClose:      afterCloseOutlives,
		lawid.PoisonConsistent:         poisonHeals,
		lawid.AppenderMonotonicOffsets: appenderFreezes,
	}
}

// differentialDefect is the rule that proves the reference comparison: a
// write acknowledged and dropped.
//
// It reddens because the reference kept what the subject threw away, and
// the differential compares the two after every call. That holds for any
// interface with a writer, whatever it writes, which is what makes this
// mechanical.
//
// It does NOT prove the bundled laws row, and the corpus is where that
// was settled: with the differential verdict off, whether anything
// notices a dropped write depends on which laws bound and which methods
// their closures drive, and 23 of 75 corpus interfaces bind a set that
// tolerates it. Proving a bundled leg means aiming a defect at one law
// and silencing its siblings, which is the saturation prover's job.
//
// The writer is the one the sequences already drive, so the defect
// breaks a method the run actually calls. False where nothing writes:
// there is no dropped write to plant.
func differentialDefect(b *Bindings) (projection.Defect, *subject.Method, bool) {
	m := writerCarrier(b)
	if m == nil {
		return nil, nil, false
	}
	return projection.AnswersAnyway{
		// The harness generator's own wording for this double, because it
		// plants the same one: two sentences for one planted statement
		// read as two different defects in a report.
		Clause: projection.Clause{Text: suite.DropsWriteClause(*m)},
		Option: projection.OptionName(b.IfaceName, m.Name),
	}, m, true
}

// afterCloseOutlives plants the after-close defect: one stamped method
// keeps working past Close.
//
// Exactly one. A double ignoring Close entirely reddens a law probing
// any method and proves nothing about whether the probe set is wide
// enough, which is what the claim is really about.
func afterCloseOutlives(b *Bindings, _ *LawBinding) (projection.Defect, *subject.Method, bool) {
	m := carrierOf(b, func(m *subject.Method) bool { return m.HasMixin(mixinAfterClose) })
	if m == nil {
		return nil, nil, false
	}
	return projection.PartialOutlive{Option: projection.OptionName(b.IfaceName, m.Name)}, m, true
}

// poisonHeals plants the un-sticky poison the law forbids: the subject
// reports the stamped sentinel once and then heals.
//
// The sentinel is the same declaration that licensed the law, so the
// defect cannot break the claim by naming a different one.
func poisonHeals(b *Bindings, _ *LawBinding) (projection.Defect, *subject.Method, bool) {
	for i := range b.Methods {
		m := &b.Methods[i]
		v, stamped := m.MixinParam(mixinAfterClose, mixinAfterCloseSentinel)
		if !stamped || v == "" {
			continue
		}
		return projection.SentinelOnce{Sentinel: projection.Expr(v)}, m, true
	}
	return nil, nil, false
}

// appenderFreezes plants the frozen position: every append lands and
// every one answers the same offset.
func appenderFreezes(b *Bindings, l *LawBinding) (projection.Defect, *subject.Method, bool) {
	// The law's own carrier, not a driven writer: this law appends
	// through a closure of its own, and the corpus's appender fixture
	// drives nothing but a reader. A defect over a method the law never
	// calls is one it cannot notice.
	m := methodNamed(b, l.carrier.Name)
	if m == nil {
		return nil, nil, false
	}
	return projection.FreezeReturn{Option: projection.OptionName(b.IfaceName, m.Name)}, m, true
}

// writerCarrier is the driven method a write-shaped defect overrides.
//
// Read off the actions rather than the methods, because a defect has to
// break something the sequences call: a writer this tier declined to
// drive is one no row would notice being wrong.
func writerCarrier(b *Bindings) *subject.Method {
	for _, a := range b.Actions {
		switch a.Shape {
		case shapeWriter, shapeAnsweringWriter, shapeCompositeWriter:
			return methodNamed(b, a.Method)
		}
	}
	return nil
}

// carrierOf is the first method the predicate accepts, nil for none.
func carrierOf(b *Bindings, match func(*subject.Method) bool) *subject.Method {
	for i := range b.Methods {
		if m := &b.Methods[i]; match(m) {
			return m
		}
	}
	return nil
}

// methodNamed resolves a driven action back to the declaration it came
// from, which is what carries the signature a defect is overridden at.
func methodNamed(b *Bindings, name string) *subject.Method {
	return carrierOf(b, func(m *subject.Method) bool { return m.Name == name })
}
