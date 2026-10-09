package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// configFileName is the file Discover looks for, nearest directory wins
// (charter §6.4.2 #1: golangci/protolint-style walk-up, no multi-level
// merging in v0.1).
const configFileName = ".vanguard.yaml"

// ruleKeyPattern accepts the two config key shapes of §6.4.1: a family
// prefix ("R1xx") or an exact rule id ("R1xx-02") — always within the
// R1xx..R6xx taxonomy bands of charter §3.0.
var ruleKeyPattern = regexp.MustCompile(`^R[1-6]xx(-[0-9]{2})?$`)

// Load reads and validates the .vanguard.yaml at path. Every schema
// violation fails with an error that cites the file (base name) and the
// line of the offending key — the exit-2 class of charter §6.3 ("config
// sai schema kèm file+dòng"). On success the returned Config carries Path
// and Dir, the anchor for every relative glob in the file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg, err := parseConfig(data, path)
	if err != nil {
		return nil, err
	}
	cfg.Path = path
	cfg.Dir = filepath.Dir(path)
	return cfg, nil
}

// Discover walks up from startDir looking for .vanguard.yaml and returns
// the nearest one (charter §6.4.2 #1). Finding nothing anywhere up the
// tree is not an error: (path, false, nil). The CLI layers `--config` /
// `--no-config` on top of this.
func Discover(startDir string) (string, bool, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", false, fmt.Errorf("resolve config search root %s: %w", startDir, err)
	}
	for {
		candidate := filepath.Join(dir, configFileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, nil
		}
		dir = parent
	}
}

// configError is the exit-2 error shape: "<config base name>:<line>: <why>".
// The base name (not the full path) keeps messages stable across how the
// tool was invoked; the full path is in Load's caller hands when needed.
type configError struct {
	file string
	line int
	msg  string
}

func (e *configError) Error() string {
	return fmt.Sprintf("%s:%d: %s", filepath.Base(e.file), e.line, e.msg)
}

// errAt builds a configError anchored at a YAML node (its line), or at
// line 1 when there is no node (missing keys, empty file).
func errAt(file string, node *yaml.Node, format string, args ...any) error {
	line := 1
	if node != nil && node.Line > 0 {
		line = node.Line
	}
	return &configError{file: file, line: line, msg: fmt.Sprintf(format, args...)}
}

// parseConfig walks the YAML tree node by node instead of decoding into a
// struct: the §6.3 error contract needs per-key line numbers, duplicate
// detection and exact wording, none of which a plain decode gives.
func parseConfig(data []byte, file string) (*Config, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		// YAML syntax error: surface it under the same file:line contract.
		return nil, errAt(file, nil, "%s", syntaxMessage(err))
	}

	cfg := &Config{Include: []string{"**"}}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, errAt(file, nil, `missing required key "version"`)
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, errAt(file, root, "config root must be a mapping, got %s", nodeKindName(root))
	}

	seen := map[string]bool{}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, val := root.Content[i], root.Content[i+1]
		if seen[key.Value] {
			return nil, errAt(file, key, "duplicate key %q", key.Value)
		}
		seen[key.Value] = true
		switch key.Value {
		case "version":
			if err := parseVersion(cfg, val, file); err != nil {
				return nil, err
			}
		case "frameworks":
			if err := parseFrameworks(cfg, val, file); err != nil {
				return nil, err
			}
		case "include", "exclude":
			globs, err := parseGlobList(val, key.Value, file)
			if err != nil {
				return nil, err
			}
			if key.Value == "include" {
				cfg.Include = globs
			} else {
				cfg.Exclude = globs
			}
		case "rules":
			if err := parseRules(cfg, val, file); err != nil {
				return nil, err
			}
		case "suppressions":
			if err := parseSuppressions(cfg, val, file); err != nil {
				return nil, err
			}
		default:
			return nil, errAt(file, key, "unknown key %q (known: version, frameworks, include, exclude, rules, suppressions)", key.Value)
		}
	}

	if !seen["version"] {
		return nil, errAt(file, nil, `missing required key "version"`)
	}
	return cfg, nil
}

// syntaxMessage flattens a yaml syntax error into the file:line contract.
// yaml.v3 spells line numbers as "yaml: line N: ..."; empty files parse
// cleanly and are handled as missing-version above.
func syntaxMessage(err error) string {
	msg := err.Error()
	if idx := strings.Index(msg, "line "); idx >= 0 {
		if rest := msg[idx+len("line "):]; len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
			return msg[idx:]
		}
	}
	return "line 1: " + msg
}

// nodeKindName names a YAML node kind for the "config root must be a
// mapping" error so a scalar/array root is reported in the user's
// vocabulary instead of a numeric kind code.
func nodeKindName(n *yaml.Node) string {
	switch n.Kind {
	case yaml.DocumentNode:
		return "a document"
	case yaml.SequenceNode:
		return "a sequence"
	case yaml.MappingNode:
		return "a mapping"
	case yaml.ScalarNode:
		return "a scalar"
	case yaml.AliasNode:
		return "an alias"
	default:
		return "an unknown node kind"
	}
}

func parseVersion(cfg *Config, val *yaml.Node, file string) error {
	if val.Kind != yaml.ScalarNode || val.Tag != "!!int" {
		return errAt(file, val, `"version" must be an integer, got %q`, val.Value)
	}
	n, err := strconv.Atoi(val.Value)
	if err != nil {
		return errAt(file, val, `"version" must be an integer, got %q`, val.Value)
	}
	cfg.Version = n
	if n != 1 {
		return errAt(file, val, "unsupported schema version %d (this build reads version 1)", n)
	}
	return nil
}

func parseFrameworks(cfg *Config, val *yaml.Node, file string) error {
	if val.Kind != yaml.MappingNode {
		return errAt(file, val, "frameworks must be a mapping of language to adapter list")
	}
	cfg.Frameworks = map[string][]string{}
	for i := 0; i+1 < len(val.Content); i += 2 {
		lang, adapters := val.Content[i], val.Content[i+1]
		if adapters.Kind != yaml.SequenceNode {
			return errAt(file, adapters, "frameworks.%s must be a list of adapter ids", lang.Value)
		}
		names := make([]string, 0, len(adapters.Content))
		for _, a := range adapters.Content {
			if a.Kind != yaml.ScalarNode {
				return errAt(file, a, "frameworks.%s must be a list of adapter ids", lang.Value)
			}
			names = append(names, a.Value)
		}
		cfg.Frameworks[lang.Value] = names
	}
	return nil
}

// parseGlobList reads include/exclude and validates every pattern at load
// time (TestValidateGlob discipline: a malformed glob must fail the config,
// not silently match nothing at scan time).
func parseGlobList(val *yaml.Node, what, file string) ([]string, error) {
	if val.Kind != yaml.SequenceNode {
		return nil, errAt(file, val, "%s must be a list of glob patterns", what)
	}
	out := make([]string, 0, len(val.Content))
	for _, item := range val.Content {
		if item.Kind != yaml.ScalarNode {
			return nil, errAt(file, item, "%s must be a list of glob patterns", what)
		}
		if err := ValidateGlob(item.Value); err != nil {
			return nil, errAt(file, item, "%s: invalid glob %q: %v", what, item.Value, err)
		}
		out = append(out, item.Value)
	}
	return out, nil
}

func parseRules(cfg *Config, val *yaml.Node, file string) error {
	if val.Kind != yaml.MappingNode {
		return errAt(file, val, "rules must be a mapping of rule id or family prefix to overrides")
	}
	cfg.Rules = map[string]RuleOverride{}
	for i := 0; i+1 < len(val.Content); i += 2 {
		key, entry := val.Content[i], val.Content[i+1]
		if !ruleKeyPattern.MatchString(key.Value) {
			return errAt(file, key, "invalid rule key %q (want a family prefix R<1-6>xx or an exact id R<1-6>xx-NN)", key.Value)
		}
		if entry.Kind != yaml.MappingNode {
			return errAt(file, entry, "rules.%s must be a mapping of overrides (severity, disabled, options)", key.Value)
		}
		ov := RuleOverride{}
		subSeen := map[string]bool{}
		for j := 0; j+1 < len(entry.Content); j += 2 {
			subKey, subVal := entry.Content[j], entry.Content[j+1]
			if subSeen[subKey.Value] {
				return errAt(file, subKey, "duplicate key %q in rules.%s", subKey.Value, key.Value)
			}
			subSeen[subKey.Value] = true
			switch subKey.Value {
			case "severity":
				sev, ok := ParseSeverity(subVal.Value)
				if !ok {
					return errAt(file, subVal, "invalid severity %q (want ERROR, WARN or INFO)", subVal.Value)
				}
				ov.Severity = sev
			case "disabled":
				if subVal.Tag != "!!bool" {
					return errAt(file, subVal, "disabled must be a boolean (true or false), got %q", subVal.Value)
				}
				b, err := strconv.ParseBool(subVal.Value)
				if err != nil {
					return errAt(file, subVal, "disabled must be a boolean (true or false), got %q", subVal.Value)
				}
				ov.Disabled = &b
			case "options":
				if subVal.Kind != yaml.MappingNode {
					return errAt(file, subVal, "options must be a mapping of option name to value")
				}
				opts := map[string]any{}
				if err := subVal.Decode(&opts); err != nil {
					return errAt(file, subVal, "options: %v", err)
				}
				ov.Options = opts
			default:
				return errAt(file, subKey, `unknown key %q (known under a rule: severity, disabled, options)`, subKey.Value)
			}
		}
		cfg.Rules[key.Value] = ov
	}
	return nil
}

func parseSuppressions(cfg *Config, val *yaml.Node, file string) error {
	if val.Kind != yaml.SequenceNode {
		return errAt(file, val, "suppressions must be a list of {rule, paths, reason}")
	}
	for _, entry := range val.Content {
		if entry.Kind != yaml.MappingNode {
			return errAt(file, entry, "each suppression must be a mapping of {rule, paths, reason}")
		}
		sup := PathSuppression{}
		var ruleNode, pathsNode *yaml.Node
		for j := 0; j+1 < len(entry.Content); j += 2 {
			key, item := entry.Content[j], entry.Content[j+1]
			switch key.Value {
			case "rule":
				if item.Kind != yaml.ScalarNode {
					return errAt(file, item, `suppression "rule" must be a rule id or family prefix`)
				}
				sup.Rule, ruleNode = item.Value, item
			case "paths":
				pathsNode = item
			case "reason":
				if item.Kind != yaml.ScalarNode {
					return errAt(file, item, "suppression reason must be a string")
				}
				sup.Reason = item.Value
			default:
				return errAt(file, key, `unknown key %q (known in a suppression: rule, paths, reason)`, key.Value)
			}
		}
		if ruleNode == nil {
			return errAt(file, entry, `suppression entry missing required key "rule"`)
		}
		if !ValidRuleRef(sup.Rule) {
			return errAt(file, ruleNode, `suppression "rule" %q is not a rule id or family prefix (R<1-6>xx[-NN])`, sup.Rule)
		}
		if pathsNode == nil {
			return errAt(file, entry, `suppression entry missing required key "paths"`)
		}
		if pathsNode.Kind != yaml.SequenceNode {
			return errAt(file, pathsNode, `suppression "paths" must be a list of glob patterns`)
		}
		sup.Paths = make([]string, 0, len(pathsNode.Content))
		for _, p := range pathsNode.Content {
			if p.Kind != yaml.ScalarNode {
				return errAt(file, p, `suppression "paths" must be a list of glob patterns`)
			}
			if err := ValidateGlob(p.Value); err != nil {
				return errAt(file, p, `suppression "paths": invalid glob %q: %v`, p.Value, err)
			}
			sup.Paths = append(sup.Paths, p.Value)
		}
		cfg.Suppressions = append(cfg.Suppressions, sup)
	}
	return nil
}

// resolvedRule is a Rule with the config applied: severity, enabled flag
// and options after the §6.4.2 #4 merge (family prefix first, exact id
// second — the exact level wins; metadata defaults fill what config left).
type resolvedRule struct {
	Disabled bool
	Severity Severity
	Options  map[string]any
}

// resolveRule applies cfg (nil = defaults) to a rule's metadata.
func resolveRule(cfg *Config, r Rule) resolvedRule {
	res := resolvedRule{Disabled: r.DefaultDisabled, Severity: r.Severity}
	if cfg == nil {
		return res
	}
	if fam, ok := cfg.Rules[familyPrefix(r.ID)]; ok {
		if fam.Severity != "" {
			res.Severity = fam.Severity
		}
		if fam.Disabled != nil {
			res.Disabled = *fam.Disabled
		}
		res.Options = copyOptions(fam.Options)
	}
	if ov, ok := cfg.Rules[r.ID]; ok {
		if ov.Severity != "" {
			res.Severity = ov.Severity
		}
		if ov.Disabled != nil {
			res.Disabled = *ov.Disabled
		}
		if len(ov.Options) > 0 {
			merged := copyOptions(res.Options)
			if merged == nil {
				merged = map[string]any{} // family left no options; copyOptions is nil then
			}
			for k, v := range ov.Options {
				merged[k] = v
			}
			res.Options = merged
		}
	}
	return res
}

// familyPrefix returns the "R1xx" prefix of a rule id; unknown shapes
// return the id itself (the map lookup simply misses).
func familyPrefix(id string) string {
	if len(id) >= 4 && id[0] == 'R' {
		return id[:4]
	}
	return id
}

func copyOptions(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
