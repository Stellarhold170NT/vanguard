// Command vanguard is the CLI entry point of the Vanguard linter.
//
// Vanguard scans multi-language source code directly, auto-detects the REST
// and gRPC API surface, and checks API design against AIP-style rules with
// file:line:col findings. The binary stays thin on purpose (charter §5.1):
// everything below the process boundary belongs to internal/cli, whose full
// cobra command surface is wired by w2-06.
package main

import (
	"os"

	"github.com/Stellarhold170NT/vanguard/internal/cli"
)

// Build identity reported by `vanguard version`. `version` stays the
// development value until the first tagged release (v0.1.0, W6); release
// tooling and `make bin` override all three via
// -ldflags "-X main.version=… -X main.commit=… -X main.date=…" (w2-06).
// Zero values degrade to dev/unknown inside internal/cli — the build never
// fails over missing metadata.
var (
	version = "0.1.0-dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}, os.Stdout, os.Stderr))
}
