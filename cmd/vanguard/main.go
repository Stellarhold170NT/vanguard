// Command vanguard is the CLI entry point of the Vanguard linter.
//
// Vanguard scans multi-language source code directly, auto-detects the REST
// and gRPC API surface, and checks API design against AIP-style rules with
// file:line:col findings. The full command surface (scan, check, explain,
// init, version) is wired by W2 (w2-06); this skeleton only proves the
// module builds and reports its version.
package main

import "fmt"

// version is the development version reported until the first tagged
// release (v0.1.0, W6). Release tooling may override it via ldflags.
var version = "0.1.0-dev"

func main() {
	fmt.Printf("vanguard %s\n", version)
}
