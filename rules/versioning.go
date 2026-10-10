package rules

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/vanguard-lint/vanguard/internal/engine"
	"github.com/vanguard-lint/vanguard/internal/ir"
)

// The R6xx versioning family (w3-06, charter §3.6, AIP-215/180 for HTTP
// paths and AIP-131..136 applied to rpc names). Every check reads only the
// IR: the Service.BasePath the Spring mapping merged, and the
// declaration-level GrpcServices the proto reader emitted. R6xx-01 is the
// charter's default-OFF policy rule: versioning is a choice, not a defect,
// so it stays silent until a config enables it.

//go:embed data/R6xx-01.yaml
var versionedPathYAML []byte

//go:embed data/R6xx-02.yaml
var grpcStandardMethodsYAML []byte

// defaultVersionPattern is the charter's default: a /vN version segment.
// It is a segment-anchored substring ("/v1" inside "/api/v1/orders"
// matches; "rev1" does not).
const defaultVersionPattern = `/v[0-9]+`

// serviceSelector matches Service nodes — R6xx-01 reasons about the base
// path of a controller (or proto service) grouping.
type serviceSelector struct{}

func (serviceSelector) Matches(node ir.Node) bool {
	_, ok := node.(ir.Service)
	return ok
}

// grpcServiceSelector matches the declaration-level proto services.
type grpcServiceSelector struct{}

func (grpcServiceSelector) Matches(node ir.Node) bool {
	_, ok := node.(ir.GrpcService)
	return ok
}

// versionPattern resolves R6xx-01's effective pattern: the configured
// versionPattern option or the default. A configured value that does not
// compile is loud — the engine turns the panic into a per-rule diagnostic
// instead of silently scanning with a broken pattern.
func versionPattern(ctx *engine.LintContext) *regexp.Regexp {
	pattern := defaultVersionPattern
	if v := optionString(ctx, "versionPattern"); v != "" {
		pattern = v
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic(fmt.Sprintf("rule R6xx-01: invalid versionPattern %q: %v", pattern, err))
	}
	return re
}

// versionedPathCheck is R6xx-01: flag a service base path that carries no
// version segment under the effective pattern. An empty base path has
// nothing to check (methods carry their own paths; the service-level
// contract is the base path the charter names).
func versionedPathCheck(ctx *engine.LintContext, node ir.Node) []engine.Finding {
	s, ok := node.(ir.Service)
	if !ok || s.BasePath == "" {
		return nil
	}
	re := versionPattern(ctx)
	if re.MatchString(s.BasePath) {
		return nil
	}
	return []engine.Finding{{
		Message: fmt.Sprintf("Base path %s carries no version segment matching %q — clients cannot pin the contract version.",
			s.BasePath, re.String()),
		Suggestion: "add a version segment to the base path (e.g. /api/v1/…), or set the R6xx-01 versionPattern option to your versioning scheme",
		Location:   s.Location,
	}}
}

// grpcStandardVerbs are the AIP-131..135 Standard Method names plus the
// batch variants and AIP-137's Search; a rpc is standard-named when its
// name starts with one of these and continues with a resource noun.
var grpcStandardVerbs = []string{
	"BatchCreate", "BatchDelete", "BatchGet", "BatchUpdate", // longest first — prefix matching
	"Get", "List", "Create", "Update", "Delete", "Search",
}

// grpcActionVerbs are the leading verbs that make a non-standard rpc a
// proper AIP-136 custom method (VerbNoun): exact first-camel-word match,
// no stemming. Deliberately narrow — a verb missing here under-reports
// instead of guessing (the archive/generate canonical AIP-136 examples are
// in; words that read as nouns first are not).
var grpcActionVerbs = map[string]bool{
	"activate": true, "approve": true, "archive": true, "assign": true,
	"cancel": true, "clone": true, "complete": true, "convert": true,
	"deactivate": true, "disable": true, "download": true, "enable": true,
	"execute": true, "export": true, "generate": true, "grant": true,
	"import": true, "invoke": true, "invite": true, "lock": true,
	"login": true, "logout": true, "merge": true, "migrate": true,
	"move": true, "notify": true, "preview": true, "process": true,
	"publish": true, "purge": true, "recalculate": true, "refresh": true,
	"register": true, "reject": true, "renew": true, "reset": true,
	"restore": true, "retry": true, "revoke": true, "send": true,
	"split": true, "submit": true, "sync": true, "transfer": true,
	"trigger": true, "undelete": true, "unlock": true, "unpublish": true,
	"upload": true, "validate": true, "verify": true,
}

// grpcCamelWords splits a camelCase rpc name into words ("SyncData" →
// sync, data); digits attach to the preceding word. Distinct from the
// field-name tokenizer other rule families use — it only ever sees rpc
// names.
func grpcCamelWords(name string) []string {
	var words []string
	start := 0
	rs := []rune(name)
	for i := 1; i < len(rs); i++ {
		if rs[i] >= 'A' && rs[i] <= 'Z' && (rs[i-1] < 'A' || rs[i-1] > 'Z' || i+1 < len(rs) && rs[i+1] >= 'a' && rs[i+1] <= 'z') {
			words = append(words, string(rs[start:i]))
			start = i
		}
	}
	words = append(words, string(rs[start:]))
	return words
}

// standardOrVerbNoun reports whether an rpc name follows the AIP naming:
// a Standard Method name (optionally suffixed with the resource noun) or a
// custom method whose leading verb is a known action and which carries a
// noun.
func standardOrVerbNoun(name string) bool {
	if name == "" {
		return false
	}
	for _, verb := range grpcStandardVerbs {
		if strings.HasPrefix(name, verb) {
			if len(name) == len(verb) {
				return true // the bare Standard Method name
			}
			rest := []rune(name[len(verb):])
			if rest[0] >= 'A' && rest[0] <= 'Z' {
				return true // Verb + Noun, e.g. GetBook
			}
		}
	}
	words := grpcCamelWords(name)
	if len(words) < 2 {
		return false // a bare word is neither Standard nor VerbNoun
	}
	return grpcActionVerbs[strings.ToLower(words[0])]
}

// grpcStandardMethodsCheck is R6xx-02: flag proto rpcs whose name is
// neither a Standard Method nor a VerbNoun custom method. The allow option
// is the documented escape hatch for a deliberate name ("mà không có lý
// do" — the reason lives in the config next to the name).
func grpcStandardMethodsCheck(ctx *engine.LintContext, node ir.Node) []engine.Finding {
	g, ok := node.(ir.GrpcService)
	if !ok {
		return nil
	}
	allow := map[string]bool{}
	for _, name := range optionStrings(ctx, "allow") {
		allow[name] = true
	}
	var findings []engine.Finding
	for _, rpc := range g.Rpcs {
		if allow[rpc] || standardOrVerbNoun(rpc) {
			continue
		}
		findings = append(findings, engine.Finding{
			Message: fmt.Sprintf("rpc %s in service %s is not a Standard Method name (Get*/List*/Create*/Update*/Delete*) or a VerbNoun custom method — the name leaks into every generated client.",
				rpc, g.Name),
			Suggestion: "rename the rpc to a Standard Method form or a <Verb><Noun> custom method (AIP-136), or allow-list the name in the R6xx-02 config with a reason",
			Location:   g.Location,
		})
	}
	return findings
}

// VersioningRules compiles the R6xx family: R6xx-01 versioned-path (WARN,
// default-disabled) and R6xx-02 grpc-standard-methods (INFO). The join is
// the production path (ParseMetadata → Compile, full §5.4 validation).
func VersioningRules() ([]engine.Rule, error) {
	records := []struct {
		yaml  []byte
		sel   engine.Selector
		check engine.CheckFunc
	}{
		{versionedPathYAML, serviceSelector{}, versionedPathCheck},
		{grpcStandardMethodsYAML, grpcServiceSelector{}, grpcStandardMethodsCheck},
	}
	out := make([]engine.Rule, 0, len(records))
	for _, rec := range records {
		meta, err := ParseMetadata(rec.yaml)
		if err != nil {
			return nil, fmt.Errorf("versioning rule metadata: %w", err)
		}
		rule, err := Compile(meta, rec.sel, rec.check)
		if err != nil {
			return nil, fmt.Errorf("compile versioning rule: %w", err)
		}
		out = append(out, rule)
	}
	return out, nil
}
