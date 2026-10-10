// Command gen-bench-repo generates the vanguard performance bench repository
// (w4-03; test strategy §5.1). It is the thin CLI half of the generator: all
// logic lives in internal/benchgen, the same split the charter uses for
// cmd/vanguard over internal/cli.
//
// Determinism contract: the same -seed and -files always produce the same
// bytes (same digest). The generated tree is measurement material — it is
// never committed (§5.1); the formula and the default seed below are.
//
// Usage (sandbox, 2cpu/4Gi, ttl long enough for the bench):
//
//	go run ./tools/gen-bench-repo -out /tmp/vanguard-bench-repo
//	go run ./tools/gen-bench-repo -out /tmp/bench -files 200   # small smoke tree
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vanguard-lint/vanguard/internal/benchgen"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("gen-bench-repo", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	out := fs.String("out", "", "target directory for the generated repository (required)")
	files := fs.Int("files", benchgen.DefaultFiles, "number of .java files to generate")
	seed := fs.Int64("seed", benchgen.DefaultSeed, "deterministic generation seed")
	force := fs.Bool("force", false, "overwrite a non-empty target directory")
	quiet := fs.Bool("quiet", false, "print only errors and the one summary line")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *out == "" {
		fmt.Fprintln(os.Stderr, "gen-bench-repo: -out is required")
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "gen-bench-repo: unexpected argument %q\n", fs.Arg(0))
		return 2
	}

	res, err := benchgen.Generate(benchgen.Plan{
		Seed:  *seed,
		Files: *files,
		Mix:   benchgen.DefaultMix,
	}, *out, *force)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-bench-repo: %v\n", err)
		return 2
	}
	if !*quiet {
		fmt.Printf("seed %d · mix %+v · %d java files (+pom.xml) · %d lines · %d bytes\ndigest %s\n",
			*seed, benchgen.DefaultMix, res.JavaFiles, res.Lines, res.Bytes, res.Digest)
	}
	return 0
}
