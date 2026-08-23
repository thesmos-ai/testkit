module go.thesmos.sh/testkit/generator

go 1.26.6

require (
	go.thesmos.sh/eidos v1.14.2-0.20260823162445-c93c242ae5cf
	go.thesmos.sh/eidos/backend/golang v1.13.4-0.20260823162445-c93c242ae5cf
	go.thesmos.sh/eidos/eidostest v1.14.1-0.20260823162445-c93c242ae5cf
	go.thesmos.sh/eidos/frontend/golang v1.14.1-0.20260823162445-c93c242ae5cf
	go.thesmos.sh/eidos/plugins v1.14.1-0.20260823162445-c93c242ae5cf
	go.thesmos.sh/testkit v0.10.0
	go.thesmos.sh/testkit/engine v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
)

// The runtime module is developed in lockstep with this one and its current
// tree is not published. go.work covers the workspace build; this replace is
// what lets `go mod tidy` resolve per-module.
replace (
	go.thesmos.sh/testkit => ../
	go.thesmos.sh/testkit/engine => ../engine
)
