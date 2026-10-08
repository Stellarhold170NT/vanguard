package main

import (
	"regexp"
	"testing"
)

// TestVersionFormat guards the version the skeleton binary reports: CI and
// (later) goreleaser rely on a stable, semver-compatible value.
func TestVersionFormat(t *testing.T) {
	semverRe := regexp.MustCompile(`^v?0\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)
	if !semverRe.MatchString(version) {
		t.Fatalf("version %q is not semver-compatible", version)
	}
}
