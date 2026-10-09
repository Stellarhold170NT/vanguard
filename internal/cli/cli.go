// Package cli implements the vanguard command-line surface (charter §6.1).
//
// Design invariants (charter §6.3, w2-06 brief):
//
//   - Exit codes: 0 = clean (WARN/INFO allowed, no API surface allowed),
//     1 = at least one ERROR finding, 2 = tool or usage error (stderr only).
//   - Streams: findings go to stdout (or --output FILE); tool errors and
//     --verbose evidence go to stderr; --format json|sarif keeps stdout
//     pure machine-readable (no banner, no ANSI).
//   - The binary layer (cmd/vanguard) stays thin: it only wires build
//     metadata and delegates to Run.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Stellarhold170NT/vanguard/internal/discovery"
	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/internal/render"
	"github.com/Stellarhold170NT/vanguard/rules"
)

// BuildInfo is the build identity the binary layer forwards in; release
// tooling overrides it via -ldflags -X (see Makefile target `bin` and
// cmd/vanguard's vars). Zero values degrade gracefully: `vanguard version`
// still prints one line (never fails the build — w2-06 brief §3).
type BuildInfo struct {
	Version string // e.g. "0.2.0"; "" → reported as "dev"
	Commit  string // short SHA; "" → "unknown"
	Date    string // RFC3339 build time; "" → "unknown"
}

// Run executes the CLI with the given arguments and build identity, writing
// help/findings to stdout and errors/evidence to stderr. The return value is
// the process exit code (§6.3): 0 clean, 1 findings, 2 misuse/tool error.
func Run(args []string, build BuildInfo, stdout, stderr io.Writer) int {
	inv := &invoker{stdout: stdout, stderr: stderr, build: build}
	if err := inv.run(args); err != nil {
		return 2
	}
	return inv.exitCode
}

// invoker carries one CLI invocation's writers, build identity, resolved
// flags and the exit code the pipeline decided on. One invoker per process
// run; tests construct partial invokers to drive single methods.
type invoker struct {
	stdout io.Writer
	stderr io.Writer
	build  BuildInfo

	// §6.2 flags (bound by addPipelineFlags; shared by scan and check —
	// only one command executes per invocation).
	format        string
	output        string
	config        string
	severity      string
	noColor       bool
	verbose       bool
	rulesSelector string
	listRulesFlag bool
	noConfig      bool
	timing        bool

	exitCode int
}

// run builds the cobra tree, executes it and prints the §6.3 error lines.
// Every error — flag parse, unknown command, tool failure — maps to exit 2
// with a `vanguard: …` message plus a --help pointer (w1-02 UX lesson).
func (inv *invoker) run(args []string) error {
	root := &cobra.Command{
		Use:   "vanguard",
		Short: "API design linter for REST and gRPC services",
		Long: `Vanguard scans source code, auto-detects the REST/gRPC API surface
and checks it against AIP-style design rules with file:line:col findings.

Exit codes (charter §6.3): 0 clean, 1 findings, 2 tool or usage error.`,
		SilenceUsage:      true,
		SilenceErrors:     true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	root.SetOut(inv.stdout)
	root.SetErr(inv.stderr)
	root.SetArgs(args)

	root.AddCommand(
		inv.newScanCmd(),
		inv.newCheckCmd(),
		inv.newExplainCmd(),
		inv.newInitCmd(),
		inv.newVersionCmd(),
	)

	cmd, err := root.ExecuteC()
	if err != nil {
		name := "vanguard"
		if cmd != nil && cmd.Name() != "vanguard" {
			name = "vanguard " + cmd.Name()
		}
		fmt.Fprintf(inv.stderr, "vanguard: %v\n", err)
		fmt.Fprintf(inv.stderr, "Run '%s --help' for usage.\n", name)
		return err
	}
	return nil
}

// newScanCmd wires `vanguard scan [path]` (§6.1/§6.2).
func (inv *invoker) newScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan [path]",
		Short: "Scan a directory tree and print the full report",
		Long: `Scan walks path (default "."), auto-detects the API surface and lints it
against the enabled rules.

Findings go to stdout, or to the --output FILE when given. --verbose evidence
(walker trail, adapter pick, skips) goes to stderr. --list-rules prints the
rule catalog and exits 0 without scanning.

Exit codes: 0 clean or no API surface, 1 findings, 2 tool/usage error.`,
		Example: `  vanguard scan .
  vanguard scan ./services/youth --format json
  vanguard scan . --format sarif -o vanguard.sarif
  vanguard scan . --rules R1xx --severity WARN
  vanguard scan . --list-rules`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			return inv.runPipeline(cmd, root, false)
		},
	}
	inv.addPipelineFlags(cmd, true)
	return cmd
}

// newCheckCmd wires `vanguard check [path]` — the CI alias of scan (§6.1):
// identical pipeline, but the default pretty output is suppressed on a
// non-TTY stdout so CI logs stay one line per run.
func (inv *invoker) newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [path]",
		Short: "CI alias of scan: same pipeline, quieter on non-TTY stdout",
		Long: `Check runs the scan pipeline against path (default ".") and reports the
verdict. On a non-TTY stdout with the default pretty format it prints only
one summary line on stderr; pass --format json|sarif or --output to get the
full machine-readable report in CI (§6.1).

Exit codes: 0 clean or no API surface, 1 findings, 2 tool/usage error.`,
		Example: `  vanguard check .                 # CI: one stderr summary line
  vanguard check . --format json   # full JSON report on stdout
  vanguard check . -o report.sarif --format sarif`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			return inv.runPipeline(cmd, root, true)
		},
	}
	inv.addPipelineFlags(cmd, false)
	return cmd
}

// addPipelineFlags binds the §6.2 flag set shared by scan and check.
func (inv *invoker) addPipelineFlags(cmd *cobra.Command, withListRules bool) {
	cmd.Flags().StringVar(&inv.format, "format", "pretty", "output format: pretty (human), json or sarif")
	cmd.Flags().StringVarP(&inv.output, "output", "o", "", "write the report to FILE instead of stdout")
	cmd.Flags().StringVar(&inv.config, "config", "", "explicit config file (skips .vanguard.yaml discovery)")
	cmd.Flags().StringVar(&inv.severity, "severity", "INFO", "display threshold: INFO, WARN or ERROR (display only)")
	cmd.Flags().BoolVar(&inv.noColor, "no-color", false, "disable ANSI color even on a terminal")
	cmd.Flags().BoolVarP(&inv.verbose, "verbose", "v", false, "evidence trail on stderr (walker, adapter pick, skips)")
	cmd.Flags().StringVar(&inv.rulesSelector, "rules", "", "comma-separated rule ids or family prefixes to enable (default: all)")
	cmd.Flags().BoolVar(&inv.noConfig, "no-config", false, "ignore config discovery; run with built-in defaults")
	// --timing opts into the engine-measured wall clock in the rendered
	// output (charter §6.6 header "files in 3.2s" / §6.7 durationMs).
	// Default OFF: the deterministic-output contract (test strategy §5.4 —
	// byte-identical runs, no wall-clock in the report bytes) applies to
	// every default invocation; CI and the golden harness never pass it.
	cmd.Flags().BoolVar(&inv.timing, "timing", false, "include the engine-measured scan duration in the report (default output is deterministic, test-strategy §5.4)")
	if withListRules {
		cmd.Flags().BoolVar(&inv.listRulesFlag, "list-rules", false, "print the sorted rule catalog with full metadata, then exit 0")
	}
}

// runPipeline is the shared scan/check body: validate → config → registry →
// discover → lint → render (charter §5.1 pipeline, CLI boundary at §6.3).
func (inv *invoker) runPipeline(cmd *cobra.Command, root string, isCheck bool) error {
	// 1. Validate the format before any work (§6.2: an unknown --format is
	// a tool error, never a render surprise).
	if _, err := render.For(render.Format(inv.format)); err != nil {
		return fmt.Errorf("%w (want pretty, json or sarif)", err)
	}
	// 2. Validate the --severity display threshold. It never changes the
	// exit code (test-strategy §5.4 #6) — only what gets displayed.
	threshold, ok := engine.ParseSeverity(inv.severity)
	if !ok {
		return fmt.Errorf("unknown severity %q (want INFO, WARN or ERROR)", inv.severity)
	}
	// 3. Config: --config, --no-config or nearest .vanguard.yaml discovery.
	cfg, cfgPath, err := inv.resolveConfig(root)
	if err != nil {
		return err
	}
	// 4. Registry: builtin rules, then the --rules selector filter.
	reg, err := registrySource()
	if err != nil {
		return fmt.Errorf("build rule registry: %w", err)
	}
	rulesTotal := len(reg.All())
	keep, err := parseRuleSelector(inv.rulesSelector, reg)
	if err != nil {
		return err
	}
	if keep != nil {
		reg, err = filterRegistry(reg, keep)
		if err != nil {
			return err
		}
	}
	// 5. --list-rules short-circuits: catalog, then exit 0 (§6.2).
	if inv.listRulesFlag {
		return inv.listRules(inv.rulesSelector)
	}
	// 6. Discovery: walk → detect → select → parse (w2-03). Discovery's
	// verbose stream already stamps the "[vanguard] " prefix (§6.3), so the
	// stderr writer is passed through untouched.
	var verboseOut io.Writer
	if inv.verbose {
		verboseOut = inv.stderr
	}
	res, err := discovery.Scan(root, discovery.ScanOptions{Verbose: verboseOut})
	if err != nil {
		return fmt.Errorf("scan %s: %w", root, err)
	}
	// 7. No API surface is a clean exit 0 with a clear notice (§6.3): on
	// the human channel for pretty, on stderr for machine formats.
	if res.NoAPI != "" {
		inv.exitCode = 0
		if render.Format(inv.format) == render.FormatPretty && inv.output == "" {
			fmt.Fprintf(inv.stdout, "vanguard: %s — nothing to check in %s.\n", res.NoAPI, root)
		} else {
			fmt.Fprintf(inv.stderr, "vanguard: %s — nothing to check in %s.\n", res.NoAPI, root)
		}
		return nil
	}
	// 8. Lint (w2-04). The engine stamps findings and applies suppression;
	// the CLI layer fills the scan-wide counters the report does not carry.
	linter := engine.NewLinter(reg)
	rep, err := linter.Run(res.Surface, cfg)
	if err != nil {
		return fmt.Errorf("lint: %w", err)
	}
	rep.FilesScanned = len(res.Files)
	rep.FilesSkipped = len(res.Skipped)
	rep.Target = root
	rep.Diagnostics = append(rep.Diagnostics, res.Diagnostics...)
	// Deterministic output by default (test strategy §5.4): the engine
	// measures real wall clock (w4-03 fixed the dead defer-start timer),
	// but the rendered report only carries it under --timing. Everything
	// downstream of this point — exit code, display, render — sees the
	// zeroed duration unless timing was explicitly requested.
	if !inv.timing {
		rep.DurationMS = 0
	}
	inv.exitCode = engine.ExitCode(rep, nil)
	// 9. check on a non-TTY stdout: suppress the default pretty report and
	// keep one summary line on stderr (§6.1). Explicit --format/--output
	// or a terminal restores the full report.
	if isCheck && !cmd.Flags().Changed("format") && inv.output == "" && !inv.tty() {
		errs, warns, infos := countSeverities(rep.Findings)
		fmt.Fprintf(inv.stderr, "vanguard check: %s — %d ERROR, %d WARN, %d INFO, %d suppressed (exit %d)\n",
			verdictWord(inv.exitCode), errs, warns, infos, rep.Suppressed, inv.exitCode)
		return nil
	}
	// 10. Render. The severity threshold filters the DISPLAY copy only —
	// inv.exitCode was already decided from the full finding set.
	display := displayReport(rep, threshold)
	infoMap := ruleInfoMap(reg)
	stats := render.ScanStats{
		RulesTotal:  rulesTotal,
		RulesActive: len(reg.All()),
	}
	if res.Surface != nil {
		stats.Services = len(res.Surface.Services)
	}
	rend, err := render.For(render.Format(inv.format),
		render.WithVersion(inv.build.versionOrDev()),
		render.WithRules(infoMap),
		render.WithStats(stats),
		render.WithConfigPath(cfgPath),
		render.WithColor(colorMode(inv.noColor)),
	)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}
	if inv.output != "" {
		f, err := os.Create(inv.output)
		if err != nil {
			return fmt.Errorf("open output %s: %w", inv.output, err)
		}
		if err := rend.Render(f, display); err != nil {
			f.Close()
			return fmt.Errorf("render: %w", err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close output %s: %w", inv.output, err)
		}
		return nil
	}
	if err := rend.Render(inv.stdout, display); err != nil {
		return fmt.Errorf("render: %w", err)
	}
	return nil
}

// resolveConfig layers the §6.2 flag pair over engine discovery (§6.4.2 #1):
// --config wins, --no-config skips, otherwise the nearest .vanguard.yaml at
// or above the scan root applies. Both flags together are a tool error.
func (inv *invoker) resolveConfig(root string) (*engine.Config, string, error) {
	if inv.config != "" && inv.noConfig {
		return nil, "", errors.New("options --config and --no-config are mutually exclusive")
	}
	if inv.config != "" {
		cfg, err := engine.Load(inv.config)
		if err != nil {
			return nil, "", err
		}
		return cfg, inv.config, nil
	}
	if inv.noConfig {
		return nil, "", nil
	}
	path, found, err := engine.Discover(root)
	if err != nil {
		return nil, "", err
	}
	if !found {
		return nil, "", nil
	}
	cfg, err := engine.Load(path)
	if err != nil {
		return nil, "", err
	}
	return cfg, path, nil
}

// defaultRegistrySource pins the registrySource seam for tests: production
// resolves to defaultRegistry (builtin rules). Tests substitute a fixture
// registry so the §6.3 exit-1 plumbing is observable end-to-end.
var registrySource = defaultRegistry

// defaultRegistry assembles the production rule set: the demo set (R6xx-99
// plus the w2-07 stub-reachable fixtures R6xx-91/92/93) and the W3 rule
// families as they land (w3-03: R1xx resources & naming; w3-04: R2xx
// methods & verbs; w3-05: R3xx pagination + R4xx payload; w3-06: R5xx
// errors + R6xx versioning). The w2-07 comment that the builtin set had
// "no stub-reachable violation" no longer holds — the R2xx verbs family
// (w3-04) fires on real HTTP surfaces of every adapter, stub included.
// All of them come from data records under rules/data joined to their
// checks by the registry (charter §3.0, §5.4).
func defaultRegistry() (*rules.Registry, error) {
	reg := rules.NewRegistry()
	for _, build := range []func() ([]engine.Rule, error){
		rules.DemoRules,
		rules.ErrorRules,
		rules.VersioningRules,
		rules.PaginationRules,
		rules.PayloadRules,
	} {
		family, err := build()
		if err != nil {
			return nil, fmt.Errorf("load builtin rules: %w", err)
		}
		for _, rule := range family {
			if err := reg.Register(rule); err != nil {
				return nil, fmt.Errorf("register %s: %w", rule.ID, err)
			}
		}
	}
	resourceRules, err := rules.ResourceRules()
	if err != nil {
		return nil, fmt.Errorf("load builtin rules: %w", err)
	}
	for _, rule := range resourceRules {
		if err := reg.Register(rule); err != nil {
			return nil, fmt.Errorf("register %s: %w", rule.ID, err)
		}
	}
	methodRules, err := rules.MethodRules()
	if err != nil {
		return nil, fmt.Errorf("load builtin rules: %w", err)
	}
	for _, rule := range methodRules {
		if err := reg.Register(rule); err != nil {
			return nil, fmt.Errorf("register %s: %w", rule.ID, err)
		}
	}
	return reg, nil
}

// parseRuleSelector parses a --rules value into the set of ids to keep:
// exact ids and family prefixes ("R6xx" keeps every R6xx-* rule), spaces
// tolerated. nil keep = everything. An empty token or an unknown reference
// is a tool error (§6.3) so a typo'd CI config fails loudly.
func parseRuleSelector(selector string, base *rules.Registry) (map[string]bool, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil, nil
	}
	keep := map[string]bool{}
	all := base.All()
	for _, tok := range strings.Split(selector, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			return nil, fmt.Errorf("empty rule reference in --rules %q", selector)
		}
		if _, ok := base.ByID(tok); ok {
			keep[tok] = true
			continue
		}
		matched := false
		for _, r := range all {
			if strings.HasPrefix(r.ID, tok+"-") {
				keep[r.ID] = true
				matched = true
			}
		}
		if !matched {
			return nil, fmt.Errorf("unknown rule %q in --rules (see 'vanguard scan --list-rules')", tok)
		}
	}
	return keep, nil
}

// filterRegistry rebuilds the registry with only the selected rules.
func filterRegistry(base *rules.Registry, keep map[string]bool) (*rules.Registry, error) {
	filtered := rules.NewRegistry()
	for _, r := range base.All() {
		if !keep[r.ID] {
			continue
		}
		if err := filtered.Register(r); err != nil {
			return nil, fmt.Errorf("re-register %s: %w", r.ID, err)
		}
	}
	return filtered, nil
}

// listRules prints the sorted, full-metadata rule catalog (§6.2 --list-rules):
// the public API for CI config review and w3-07 doc generation.
func (inv *invoker) listRules(selector string) error {
	_ = selector // the catalog always lists the full sorted set
	reg, err := registrySource()
	if err != nil {
		return err
	}
	all := reg.All()
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	tw := tabwriter.NewWriter(inv.stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "RULE ID\tSLUG\tCATEGORY\tSEVERITY\tDOC\tSUMMARY")
	for _, r := range all {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			r.ID, r.Slug, r.Category, string(r.Severity), r.DocPath, r.Summary)
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("write rule catalog: %w", err)
	}
	fmt.Fprintf(inv.stdout, "\n%d rule(s) registered. --rules accepts exact ids or family prefixes (R6xx).\n", len(all))
	return nil
}

// ruleInfoMap feeds the machine-readable renderers the rule metadata the
// engine report does not carry (charter §6.7/§6.8).
func ruleInfoMap(reg *rules.Registry) map[string]render.RuleInfo {
	out := make(map[string]render.RuleInfo, len(reg.All()))
	for _, r := range reg.All() {
		out[r.ID] = render.RuleInfo{
			ID:              r.ID,
			Slug:            r.Slug,
			Summary:         r.Summary,
			Description:     r.Summary,
			HelpURI:         r.DocPath,
			Category:        r.Category,
			DefaultSeverity: r.Severity,
		}
	}
	return out
}

// displayReport returns the report copy whose findings honour the --severity
// display threshold; the original report (and the exit code) stay untouched.
func displayReport(rep *engine.Report, threshold engine.Severity) *engine.Report {
	disp := *rep
	if severityRank(threshold) > severityRank(engine.SeverityInfo) {
		disp.Findings = filterByThreshold(rep.Findings, threshold)
	}
	return &disp
}

func filterByThreshold(findings []engine.Finding, threshold engine.Severity) []engine.Finding {
	kept := make([]engine.Finding, 0, len(findings))
	for _, f := range findings {
		if severityRank(f.Severity) >= severityRank(threshold) {
			kept = append(kept, f)
		}
	}
	return kept
}

// severityRank orders the §3.3 severities for threshold comparisons.
func severityRank(s engine.Severity) int {
	switch s {
	case engine.SeverityError:
		return 2
	case engine.SeverityWarn:
		return 1
	default:
		return 0
	}
}

func countSeverities(findings []engine.Finding) (errs, warns, infos int) {
	for _, f := range findings {
		switch f.Severity {
		case engine.SeverityError:
			errs++
		case engine.SeverityWarn:
			warns++
		default:
			infos++
		}
	}
	return errs, warns, infos
}

func verdictWord(exitCode int) string {
	if exitCode == 0 {
		return "clean"
	}
	return "findings"
}

// colorMode maps the --no-color flag onto the renderer's color contract:
// explicit never beats everything, otherwise auto (TTY && !NO_COLOR).
func colorMode(noColor bool) render.ColorMode {
	if noColor {
		return render.ColorNever
	}
	return render.ColorAuto
}

// ttyDetector is the process TTY probe; tests override it (helpers_test.go
// simulateTTY) to exercise both §6.1 check branches hermetically.
var ttyDetector = defaultTTYDetector

func defaultTTYDetector() bool {
	st, err := os.Stdout.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func (inv *invoker) tty() bool { return ttyDetector() }

// newExplainCmd wires `vanguard explain <rule-id>` (§6.1): metadata,
// examples and the doc pointer for one rule.
func (inv *invoker) newExplainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "explain <rule-id>",
		Short: "Explain one rule: metadata, examples, doc pointer",
		Long: `Explain prints the full metadata of one rule — severity, category,
documentation path and the good/bad examples from its YAML record.

An unknown id is a tool error (exit 2); run 'vanguard scan --list-rules'
for the catalog.`,
		Example: `  vanguard explain R6xx-99`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return inv.runExplain(args)
		},
	}
}

func (inv *invoker) runExplain(args []string) error {
	id := strings.TrimSpace(args[0])
	reg, err := registrySource()
	if err != nil {
		return err
	}
	rule, ok := reg.ByID(id)
	if !ok {
		return fmt.Errorf("unknown rule %q — run 'vanguard scan --list-rules' for the catalog", id)
	}
	fmt.Fprintf(inv.stdout, "%s — %s\n\n", rule.ID, rule.Summary)
	fmt.Fprintf(inv.stdout, "Slug:     %s\nCategory: %s\nSeverity: %s\nDocs:     %s\n\n",
		rule.Slug, rule.Category, string(rule.Severity), rule.DocPath)
	fmt.Fprintf(inv.stdout, "Good:\n  %s\n\nBad:\n  %s\n",
		strings.TrimRight(rule.ExampleGood, "\n"), strings.TrimRight(rule.ExampleBad, "\n"))
	return nil
}

// initTemplate is the commented .vanguard.yaml skeleton `vanguard init`
// writes; what it writes must load as a valid v0.1 config (engine.Load).
const initTemplate = `# Vanguard configuration (v%[1]s) — API design linter.
# Docs: docs/config.md. Every key below is valid v0.1 syntax; delete what
# you do not use. Globs are relative to this file's directory (§6.4.1).
version: 1

# include / exclude — which files the walker may lint (§6.4.1).
include:
  - "**"
exclude: []

# rules — per-rule overrides: disabled, severity, options (§6.4.2).
rules: {}

# suppressions — path-scoped silences, each with a reason (§5.4).
suppressions: []
`

// writeInitConfig writes the commented template into dir/.vanguard.yaml and
// refuses to overwrite an existing config (§6.1: init never clobbers).
func writeInitConfig(dir, version string) (string, error) {
	path := filepath.Join(dir, ".vanguard.yaml")
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("refusing to overwrite %s: config already exists", path)
	}
	if err := os.WriteFile(path, []byte(fmt.Sprintf(initTemplate, version)), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

// newInitCmd wires `vanguard init [dir]` (§6.1).
func (inv *invoker) newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [dir]",
		Short: "Write a commented .vanguard.yaml template",
		Long: `Init writes a commented .vanguard.yaml into dir (default "."). An existing
config is never overwritten — init refuses with an error (exit 2).`,
		Example: `  vanguard init
  vanguard init ./services/youth`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			path, err := writeInitConfig(dir, inv.build.versionOrDev())
			if err != nil {
				return err
			}
			fmt.Fprintf(inv.stdout, "wrote %s — edit include/exclude and rule overrides to taste.\n", path)
			return nil
		},
	}
}

// newVersionCmd wires `vanguard version` (§6.1): one machine-friendly line,
// never fails, degrades to dev/unknown without build metadata.
func (inv *invoker) newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, commit and build date",
		Long: `Version prints one line: "vanguard VERSION (commit SHA, built DATE)".
Without -ldflags build metadata it reports dev/unknown/unknown and still
exits 0 (w2-06 brief: never fail the build).`,
		Example: `  vanguard version`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(inv.stdout, "vanguard %s (commit %s, built %s)\n",
				inv.build.versionOrDev(), orUnknown(inv.build.Commit), orUnknown(inv.build.Date))
			return nil
		},
	}
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

func (b BuildInfo) versionOrDev() string {
	if b.Version == "" {
		return "dev"
	}
	return b.Version
}
