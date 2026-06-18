package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTargetKeepsExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app")
	if err := os.WriteFile(path, []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := resolveTarget(path); got != path {
		t.Fatalf("resolveTarget(existing file) = %q, want %q", got, path)
	}
}

func TestResolveTargetLooksUpPathCommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exex-test-command")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	if got := resolveTarget("exex-test-command"); got != path {
		t.Fatalf("resolveTarget(PATH command) = %q, want %q", got, path)
	}
}

func TestResolveTargetPassesThroughUnresolvedPath(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	if got := resolveTarget(missing); got != missing {
		t.Fatalf("resolveTarget(missing path) = %q, want %q", got, missing)
	}
	if got := resolveTarget(dir); got != dir {
		t.Fatalf("resolveTarget(directory) = %q, want %q", got, dir)
	}
}
