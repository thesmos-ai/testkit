// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package subject_test

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/ttl"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// The parameter keys come off the live registry, never a transcription.
//
// A mixin gaining a parameter has to reach the rules that condition on
// it. Transcribed here, that gain would be a silent no-op until somebody
// noticed the rule stopped firing — and a rule that stops firing emits
// nothing and reports nothing.
func TestMixinParamKeysReadTheRegistry(t *testing.T) {
	t.Parallel()

	keys := subject.MixinParamKeys(ttl.Name)
	testkit.Contains(t, keys, ttl.ParamNotFound,
		"ttl's declared parameters come from ttl, not from a list beside it")

	testkit.Len(t, subject.MixinParamKeys("not a mixin"), 0,
		"a word the registry does not know declares no parameters")
}

// A method's own mixin stamps are collected under the key tiers'
// conditions read them by.
func TestLawParamsCollectsMixinStamps(t *testing.T) {
	t.Parallel()

	m := stamped(t, map[string]string{
		ttl.Name + "." + ttl.ParamNotFound: "ErrExpired",
	})
	m.Mixins = []string{ttl.Name}

	got := subject.LawParams([]subject.Method{m}, m)

	key := shape.MixinParamKey(ttl.Name, ttl.ParamNotFound).Name()
	testkit.Equal(t, got[key], "ErrExpired",
		"keyed as the condition spells it, through the key's own Name")
}

// A method carrying no source, or no stamps, yields nothing rather than
// an empty map somebody has to guard.
func TestLawParamsOnAnUnstampedMethod(t *testing.T) {
	t.Parallel()

	m := stamped(t, nil)
	m.Mixins = []string{ttl.Name}
	testkit.Len(t, subject.LawParams([]subject.Method{m}, m), 0,
		"a mixin declaring no parameter value contributes none")

	// Sig.Source is nil for a projection that came from the emit side
	// rather than from source, which is the state both readers guard.
	emitSide := subject.Method{Sig: &golang.Sig{Name: "Get"}, Mixins: []string{ttl.Name}}
	testkit.Len(t, subject.LawParams([]subject.Method{emitSide}, emitSide), 0,
		"a projection with no declaration behind it has no stamps to read")
}
