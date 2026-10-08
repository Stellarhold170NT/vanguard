// Package cli wires the vanguard command surface — scan, check, explain,
// init, version — with cobra (charter §6.1; implementation: w2-06).
//
// Until w2-06 lands the cobra command tree, Run is the stub cmd/vanguard
// calls so the module already carries its binary contract (R4: static
// binary, runs from any cwd). Run returns the §6.3 exit code; the caller
// owns the process exit.
package cli

import (
	"fmt"
	"io"
)

// Run executes vanguard with args and returns the process exit code per the
// charter §6.3 contract: 0 = success (once scanning lands: no ERROR
// findings), 1 = at least one ERROR finding, 2 = tool/config misuse.
//
// Stub behavior, replaced by w2-06: no args or "version" prints the version
// banner; any other command is unknown and reported on stderr with exit
// code 2.
func Run(args []string, version string, stdout, stderr io.Writer) int {
	switch {
	case len(args) == 0 || args[0] == "version":
		fmt.Fprintf(stdout, "vanguard %s\n", version)
	default:
		fmt.Fprintf(stderr, "vanguard: unknown command %q — command surface arrives in w2-06 (charter §6.1)\n", args[0])
		return 2
	}
	return 0
}
