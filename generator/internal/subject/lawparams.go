// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package subject

import (
	"slices"

	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"
)

// LawParams collects the stamp parameters tiers' When clauses read,
// keyed as tiers' own Condition.Param spells them — through the eidos
// keys' own Name method, so the spelling has one home. Mixin params
// come off the method; contract params off every carrier of the
// contract, because a protocol's parameter lives on the directive
// host and a rule selected from another role conditions on it all
// the same. Shared with the model generator for the same reason
// [Method.Classifications] is.
func LawParams(methods []Method, m Method) map[string]string {
	if m.Source == nil {
		return nil
	}
	out := map[string]string{}
	for _, name := range m.Mixins {
		for _, p := range MixinParamKeys(name) {
			if v, ok := shape.MixinParamKey(name, p).Get(m.Source.Meta()); ok {
				out[shape.MixinParamKey(name, p).Name()] = v
			}
		}
	}
	for _, name := range m.Contracts {
		for _, p := range contractParamKeys(name) {
			for _, carrier := range methods {
				if carrier.Source == nil || !slices.Contains(carrier.Contracts, name) {
					continue
				}
				if v, ok := shape.ContractParamKey(name, p).Get(carrier.Source.Meta()); ok && v != "" {
					out[shape.ContractParamKey(name, p).Name()] = v
				}
			}
		}
	}
	return out
}

// MixinParamKeys names one mixin's declared parameter keys, read off
// the live registry rather than transcribed.
//
// Exported because both tiers ask it, and a transcription would be a
// second home for a vocabulary eidos already owns: a mixin gaining a
// parameter would then be a silent no-op here until somebody noticed.
func MixinParamKeys(name string) []string {
	for _, m := range mixins.All() {
		if m.Name == name {
			return shape.ParamKeys(m.Params)
		}
	}
	return nil
}

// contractParamKeys is [MixinParamKeys] on the contract axis.
func contractParamKeys(name string) []string {
	for _, c := range contracts.All() {
		if c.Name == name {
			return shape.ParamKeys(c.Params)
		}
	}
	return nil
}
