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

// version is the development version reported until the first tagged
// release (v0.1.0, W6). Release tooling overrides it via
// -ldflags "-X main.version=…"; w2-06 forwards it into internal/cli.
var version = "0.1.0-dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], version, os.Stdout, os.Stderr))
}
