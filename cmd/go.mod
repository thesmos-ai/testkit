module go.thesmos.sh/testkit/cmd

go 1.26.6

require (
	github.com/spf13/cobra v1.10.2
	go.thesmos.sh/eidos v1.14.2-0.20260823202727-3787d61fa3a8
	go.thesmos.sh/eidos/cli v1.13.4-0.20260823202727-3787d61fa3a8
	go.thesmos.sh/testkit v0.10.0
	go.thesmos.sh/testkit/generator v0.0.0
)

require (
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	go.thesmos.sh/eidos/backend/golang v1.13.4-0.20260823202727-3787d61fa3a8 // indirect
	go.thesmos.sh/eidos/frontend/golang v1.14.1-0.20260823202727-3787d61fa3a8 // indirect
	go.thesmos.sh/eidos/plugins v1.14.1-0.20260823202727-3787d61fa3a8 // indirect
	go.thesmos.sh/testkit/engine v0.0.0-00010101000000-000000000000 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
)

// The runtime module is developed in lockstep with this one and its current
// tree is not published. go.work covers the workspace build; this replace is
// what lets `go mod tidy` resolve per-module.
replace go.thesmos.sh/testkit => ../

replace go.thesmos.sh/testkit/generator => ../generator

replace go.thesmos.sh/testkit/engine => ../engine
