package discovery

import (
	"fmt"
	"path"
	"sort"
)

// buildFileLang maps build-file names to the language they pin (charter
// §5.3, verbatim list). This map is the extension point for future
// languages: add the file name here and the extension below, nothing else.
var buildFileLang = map[string]string{
	"pom.xml":          "java",
	"build.gradle":     "java",
	"go.mod":           "go",
	"requirements.txt": "python",
	"pyproject.toml":   "python",
}

// sourceExtLang maps source extensions to the language they hint (charter
// §5.3: detect "theo extension + build file").
var sourceExtLang = map[string]string{
	".java": "java",
	".go":   "go",
	".py":   "python",
}

// Signal weights: a build file is rare and decisive, one source file is a
// weak hint. The 100:1 gap makes a single pom.xml outrank any number of
// stray source files of another language.
const (
	buildFileScore  = 100
	sourceFileScore = 1
)

// Detect ranks language candidates for a walked file set using file names
// only: build files (pom.xml/build.gradle → java, go.mod → go,
// requirements.txt/pyproject.toml → python) and source extensions (.java,
// .go, .py — charter §5.3). Candidates come back best-first, score
// descending, ties broken by language name — repeated detection on the same
// input is byte-identical (charter B-6).
//
// Ranking: score = 100 per build file + 1 per source file. Every candidate
// carries Evidence naming its signals, so verbose mode can show why a
// language won or lost (charter §5.3: "chọn sai/sparse → evidence trong
// verbose").
//
// Two deliberate limits, both consequences of the w2-01 contract that
// Detect receives paths only:
//   - Candidate.Framework stays empty in v0.1: pinning a framework needs
//     file CONTENT (imports, annotations), and Detect deliberately does not
//     read files — content work is the adapter's job (w3-01 fills
//     ir.Source.Framework from the sources it parses).
//   - No signals → empty result (never nil-vs-empty ambiguity, never an
//     error): "no language found" is a normal outcome.
func Detect(files []string) []Candidate {
	buildHits := make(map[string][]string) // lang → build files found
	sourceCounts := make(map[string]int)   // lang → source file count
	extCounts := make(map[string]map[string]int)
	for _, f := range files {
		base := path.Base(f)
		if lang, ok := buildFileLang[base]; ok {
			buildHits[lang] = append(buildHits[lang], f)
			continue
		}
		ext := path.Ext(base)
		if lang, ok := sourceExtLang[ext]; ok {
			sourceCounts[lang]++
			if extCounts[lang] == nil {
				extCounts[lang] = make(map[string]int)
			}
			extCounts[lang][ext]++
		}
	}

	type scored struct {
		lang  string
		score int
	}
	var langs []scored
	for lang, hits := range buildHits {
		langs = append(langs, scored{lang, buildFileScore*len(hits) + sourceCounts[lang]*sourceFileScore})
	}
	for lang, n := range sourceCounts {
		if _, hasBuild := buildHits[lang]; !hasBuild {
			langs = append(langs, scored{lang, n * sourceFileScore})
		}
	}
	sort.Slice(langs, func(i, j int) bool {
		if langs[i].score != langs[j].score {
			return langs[i].score > langs[j].score
		}
		return langs[i].lang < langs[j].lang
	})

	candidates := make([]Candidate, 0, len(langs))
	for _, l := range langs {
		e := Evidence{}
		if hits := buildHits[l.lang]; len(hits) > 0 {
			e.Reason = fmt.Sprintf("%d build file(s) and %d source file(s) match %s",
				len(hits), sourceCounts[l.lang], l.lang)
			for _, hit := range hits {
				e.Details = append(e.Details, "build file: "+hit)
			}
		} else {
			e.Reason = fmt.Sprintf("%d source file(s) match %s (no build file)", sourceCounts[l.lang], l.lang)
		}
		for _, ext := range sortedKeys(extCounts[l.lang]) {
			e.Details = append(e.Details, fmt.Sprintf("%d × %s", extCounts[l.lang][ext], ext))
		}
		candidates = append(candidates, Candidate{Lang: l.lang, Evidence: e})
	}
	return candidates
}

// sortedKeys returns the map keys sorted, for deterministic evidence lines.
func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
