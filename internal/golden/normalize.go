package golden

import (
	"os"
	"regexp"
	"sort"
	"strings"
)

// Normalize strips exactly the two noise sources a scan output can never
// control — wall-clock duration and the absolute paths of the machine the
// test runs on — and leaves every other byte untouched (test strategy
// §5.4: determinism is never "patched" beyond these; output that varies in
// any other byte is an engine/render bug, not a harness concern).
//
// subs maps absolute path prefixes to stable tokens, e.g. the absolute
// case directory ("<case>") and the module root ("<module>").
func Normalize(data []byte, subs map[string]string) []byte {
	s := string(data)
	// Longest prefix first: the module root is a prefix of the case dir.
	keys := make([]string, 0, len(subs))
	for k := range subs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, k := range keys {
		if k != "" {
			s = strings.ReplaceAll(s, k, subs[k])
		}
	}
	s = prettyDurationRe.ReplaceAllString(s, "files in <duration>")
	s = jsonDurationRe.ReplaceAllString(s, `"durationMs": 0`)
	return []byte(s)
}

// prettyDurationRe matches the pretty header's "N files in 42ms|1.3s"
// segment (§6.6 line 2) — the one wall-clock value in the output.
var prettyDurationRe = regexp.MustCompile(`files in \d+(\.\d+)?(ms|s)`)

// jsonDurationRe matches the §6.7 summary duration field.
var jsonDurationRe = regexp.MustCompile(`"durationMs": \d+`)

// Substitutions returns the path→token map for one case: the absolute case
// directory and the absolute module root (a discovery config path in the
// pretty header is the one field that legitimately reaches stdout as an
// absolute path — everything else in the output is already relative).
func Substitutions(caseDir, moduleRoot string) map[string]string {
	return map[string]string{
		caseDir + string(os.PathSeparator):    "<case>/",
		caseDir:                               "<case>",
		moduleRoot + string(os.PathSeparator): "<module>/",
		moduleRoot:                            "<module>",
	}
}
