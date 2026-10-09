package rules

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Metadata is the data-driven record of one rule (charter §3.0 "Metadata
// là data"): a YAML record under rules/, embedded into the binary at build
// time, joined to a Go check by the registry. A rule is data — adding one
// never means editing the engine.
type Metadata struct {
	ID       string // R<F>xx-NN
	Slug     string // kebab-case, e.g. "no-verb-path"
	Category string // family category, e.g. "resources"

	// Severity is the raw spelling from the record ("ERROR"/"WARN"/"INFO",
	// any case); Compile validates and converts it to the engine constant.
	Severity string

	Summary     string // one-line human description
	DocPath     string // documentation page path (w3-07 owns the files)
	ExampleGood string // compliant example, rendered by explain/docs
	ExampleBad  string // violating example

	// DefaultDisabled marks opt-in policy rules (charter §3.6).
	DefaultDisabled bool

	// Options lists the option keys the check reads; config may only set
	// declared keys (engine enforces, exit 2).
	Options []string
}

// metadataRecord mirrors Metadata for strict YAML decoding: unknown keys
// are rejected so a typo'd record fails its test instead of silently
// dropping a field (charter §3.0 "Metadata là data" + §6.3 error
// discipline). yaml.v3 reports the offending line number in the error.
type metadataRecord struct {
	ID              string   `yaml:"id"`
	Slug            string   `yaml:"slug"`
	Category        string   `yaml:"category"`
	Severity        string   `yaml:"severity"`
	Summary         string   `yaml:"summary"`
	DocPath         string   `yaml:"docPath"`
	ExampleGood     string   `yaml:"exampleGood"`
	ExampleBad      string   `yaml:"exampleBad"`
	DefaultDisabled bool     `yaml:"defaultDisabled"`
	Options         []string `yaml:"options"`
}

// ParseMetadata parses one rule's YAML record. Required keys: id, slug,
// severity; everything else is optional (TestParseMetadataMinimal). The
// error always carries the YAML line of the offending key so a bad record
// is fixable at a glance.
func ParseMetadata(data []byte) (Metadata, error) {
	var rec metadataRecord
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&rec); err != nil {
		return Metadata{}, fmt.Errorf("rule metadata: %w", err)
	}
	return Metadata(rec), nil
}
