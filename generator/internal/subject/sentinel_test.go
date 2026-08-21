// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package subject_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/notfound"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/ttl"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// Two declarations can name a miss sentinel, and which one wins is the
// whole of this function.
//
// Expiry and absence are separate conditions that usually coincide. A
// store whose lapsed reads answer differently from its missing ones says
// so on ttl; one where they agree declares only notfound and falls
// through to it. Taking the first stamp found instead — which an earlier
// version did — hands an interface stamping ttl beside another
// sentinel-carrying mixin whichever the walk reached first.
func TestMissSentinel(t *testing.T) {
	t.Parallel()

	t.Run("notfound is the ordinary answer", func(t *testing.T) {
		t.Parallel()
		m := stamped(t, map[string]string{
			notfound.Name + "." + notfound.ParamSentinel: "ErrNotFound",
		})
		got, declared := subject.MissSentinel(m)
		testkit.True(t, declared, "the read declares its own miss")
		testkit.Equal(t, got, "ErrNotFound", "and it is what notfound named")
	})

	t.Run("ttl overrides it where both are declared", func(t *testing.T) {
		t.Parallel()
		// The precedence is the point: a lapsed read is not a missing one
		// for a subject that distinguishes them, and the author said which.
		m := stamped(t, map[string]string{
			notfound.Name + "." + notfound.ParamSentinel: "ErrNotFound",
			ttl.Name + "." + ttl.ParamNotFound:           "ErrExpired",
		})
		got, declared := subject.MissSentinel(m)
		testkit.True(t, declared, "one of the two declares it")
		testkit.Equal(t, got, "ErrExpired", "and ttl's answer wins")
	})

	t.Run("a method declaring neither reports none", func(t *testing.T) {
		t.Parallel()
		_, declared := subject.MissSentinel(stamped(t, nil))
		testkit.False(t, declared, "nothing named a sentinel")
	})

	t.Run("a projection from the emit side reports none", func(t *testing.T) {
		t.Parallel()
		// Sig.Source is nil where the projection did not come from source,
		// and the reader must not go looking for a bag that never existed.
		_, declared := subject.MissSentinel(subject.Method{Sig: &golang.Sig{Name: "Get"}})
		testkit.False(t, declared, "no declaration, no stamp")
	})
}

// stamped builds a method whose source carries the given mixin params,
// keyed `<mixin>.<param>`.
func stamped(t *testing.T, params map[string]string) subject.Method {
	t.Helper()
	src := storefixture.New().
		Package("kv", "example.com/kv").
		Interface("Store", func(b *storefixture.InterfaceBuilder) {
			b.Pos(sdk.At("kv/iface.go", 1, 1))
			b.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Pos(sdk.At("kv/iface.go", 2, 1))
			})
		}).
		Build()

	var method *sdk.Method
	for _, iface := range src.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			method = m
		}
	}
	if method == nil {
		t.Fatal("fixture declares no method")
	}
	for key, v := range params {
		mixin, param := splitKey(t, key)
		shape.MixinParamKey(mixin, param).Set(method.EnsureMeta(), v, "test")
	}
	return subject.Method{Sig: &golang.Sig{Name: method.Name, Source: method}}
}

// splitKey cuts a `<mixin>.<param>` fixture key.
func splitKey(t *testing.T, key string) (mixin, param string) {
	t.Helper()
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '.' {
			return key[:i], key[i+1:]
		}
	}
	t.Fatalf("fixture key %q names no param", key)
	return "", ""
}
