// Package integration holds slow, toolchain-dependent end-to-end tests guarded
// by the "crosscompile" build tag (see cross_test.go). Without that tag the
// package is intentionally empty, so the default `go test ./...` skips it and
// stays fast and hermetic.
package integration
