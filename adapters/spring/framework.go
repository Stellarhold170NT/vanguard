package spring

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Stellarhold170NT/vanguard/adapters/java"
)

// maxReadBytes bounds ONE build-file/spec read — the walker's published
// cap (internal/discovery's DefaultMaxFileSize, 1 MiB) mirrored locally:
// this package cannot import internal/discovery (its registry is the
// composition root that wires this adapter in, which would cycle). The
// walker enforces the same cap independently; this is defense in depth.
const maxReadBytes = 1 << 20 // 1 MiB

// frameworkInfo is what the build files and the parsed sources say about
// the framework: the normalized id, its version when a build file
// declares one, and whether springdoc rides along (the overlay's other
// trigger besides a committed spec file).
type frameworkInfo struct {
	framework string
	version   string
	springdoc bool
}

// springAnnotations are the org.springframework marker annotations the
// source scan reads as a framework signal (web model + core stereotypes).
// JPA annotations do not count: a repo can use Hibernate without Spring.
var springAnnotations = map[string]bool{
	"RestController": true, "Controller": true, "ControllerAdvice": true,
	"RestControllerAdvice": true, "RequestMapping": true,
	"GetMapping": true, "PostMapping": true, "PutMapping": true,
	"PatchMapping": true, "DeleteMapping": true, "PathVariable": true,
	"RequestParam": true, "RequestBody": true, "RequestHeader": true,
	"ResponseStatus": true, "ExceptionHandler": true, "ResponseBody": true,
	"CrossOrigin": true, "SpringBootApplication": true,
	"EnableAutoConfiguration": true, "Component": true, "Service": true,
	"Repository": true, "Autowired": true, "Configuration": true,
	"Bean": true, "Validated": true, "Transactional": true,
	"ControllerAdviceBean": true,
}

// detectFramework pins Source.Framework: "spring-boot" when the sources
// carry Spring annotations or a build file declares Spring Boot, with the
// version read from the first build file in the walked set that declares
// one (deterministic by input order). Reading is bounded and failures are
// silent — a build file we cannot read costs nothing but the version.
func detectFramework(root string, files []string, res *java.Result) frameworkInfo {
	info := frameworkInfo{}
	if res != nil && resultHasSpringAnnotation(res) {
		info.framework = "spring-boot"
	}
	for _, f := range files {
		base := path.Base(f)
		if base != "pom.xml" && base != "build.gradle" && base != "build.gradle.kts" {
			continue
		}
		data, err := readBounded(root, f)
		if err != nil {
			continue
		}
		var bi frameworkInfo
		if base == "pom.xml" {
			bi = frameworkFromPom(data)
		} else {
			bi = frameworkFromGradle(data)
		}
		if bi.framework != "" {
			if info.framework == "" {
				info.framework = bi.framework
			}
			if info.version == "" {
				info.version = bi.version
			}
		}
		if bi.springdoc {
			info.springdoc = true
		}
	}
	return info
}

// resultHasSpringAnnotation scans every annotation the extraction saw —
// classes, methods, parameters, fields — for a Spring marker.
func resultHasSpringAnnotation(res *java.Result) bool {
	for _, f := range res.Files {
		var walk func(cls *java.Class) bool
		walk = func(cls *java.Class) bool {
			for _, a := range cls.Annotations {
				if springAnnotations[a.Name] {
					return true
				}
			}
			for _, m := range cls.Methods {
				if annsHaveSpring(m.Annotations) {
					return true
				}
				for _, p := range m.Params {
					if annsHaveSpring(p.Annotations) {
						return true
					}
				}
			}
			for _, fl := range cls.Fields {
				if annsHaveSpring(fl.Annotations) {
					return true
				}
			}
			for _, n := range cls.Nested {
				if walk(n) {
					return true
				}
			}
			return false
		}
		for _, c := range f.Types {
			if walk(c) {
				return true
			}
		}
	}
	return false
}

func annsHaveSpring(anns []java.Annotation) bool {
	for _, a := range anns {
		if springAnnotations[a.Name] {
			return true
		}
	}
	return false
}

// readBounded loads one file relative to the scan root, refusing escape
// paths and oversized content (the adapter read contract, mirrored for
// the non-Java files this package reads).
func readBounded(root, rel string) ([]byte, error) {
	if rel == ".." || strings.HasPrefix(rel, "../") || path.IsAbs(rel) {
		return nil, fmt.Errorf("path %q escapes the scan root — refused", rel)
	}
	f, err := os.Open(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", rel, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxReadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", rel, err)
	}
	if len(data) > maxReadBytes {
		return nil, fmt.Errorf("%s exceeds the %d-byte read cap — file skipped", rel, maxReadBytes)
	}
	return data, nil
}

// frameworkFromPom reads the framework out of a pom.xml: the
// spring-boot-starter-parent version, then the spring-boot.version
// property, then an explicit org.springframework.boot dependency version.
// springdoc dependencies set the overlay trigger.
func frameworkFromPom(data []byte) frameworkInfo {
	info := frameworkInfo{}
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	var stack []string
	var groupID, artifactID, text string
	flush := func() {
		switch last(stack) {
		case "groupId":
			groupID = strings.TrimSpace(text)
		case "artifactId":
			artifactID = strings.TrimSpace(text)
		case "version":
			v := strings.TrimSpace(text)
			if groupID == "org.springframework.boot" && info.version == "" {
				info.framework, info.version = "spring-boot", v
			}
		}
		text = ""
	}
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			stack = append(stack, t.Name.Local)
			text = ""
		case xml.EndElement:
			flush()
			if t.Name.Local == "dependency" && artifactID != "" {
				if strings.HasPrefix(artifactID, "springdoc-openapi") {
					info.springdoc = true
				}
				groupID, artifactID = "", ""
			}
			if t.Name.Local == "parent" {
				groupID, artifactID = "", ""
			}
			stack = stack[:max(0, len(stack)-1)] // Go 1.21 builtin max
		case xml.CharData:
			text += string(t)
		}
	}
	// properties block: <spring-boot.version>3.2.5</spring-boot.version>
	if info.version == "" {
		if v := pomProperty(data, "spring-boot.version"); v != "" {
			info.framework, info.version = "spring-boot", v
		}
	}
	return info
}

// pomProperty reads one <properties> entry from raw pom text (the token
// walker above tracks dependencies/parents; properties are simpler as a
// literal scan).
func pomProperty(data []byte, name string) string {
	tag := "<" + name + ">"
	i := strings.Index(string(data), tag)
	if i < 0 {
		return ""
	}
	rest := string(data)[i+len(tag):]
	j := strings.Index(rest, "</"+name+">")
	if j < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:j])
}

func last(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[len(s)-1]
}

// gradleBootVersion matches `id 'org.springframework.boot' version 'x'`
// (groovy and kotlin DSL spellings).
var gradleBootVersion = regexp.MustCompile(`(?m)org\.springframework\.boot['"]?\s*(?:version\s+|=?\s+)['"]([0-9][^'"]*)['"]`)

// gradleBootVariable matches `springBootVersion = 'x'` style properties.
var gradleBootVariable = regexp.MustCompile(`(?m)(?:springBoot|spring_boot|springBootVersion)\w*\s*=?\s*['"]([0-9][^'"]*)['"]`)

// gradleSpringdoc matches springdoc coordinates in dependency strings.
var gradleSpringdoc = regexp.MustCompile(`(?m)springdoc-openapi`)

// frameworkFromGradle reads build.gradle / build.gradle.kts: the plugin
// id version first, then a springBootVersion property.
func frameworkFromGradle(data []byte) frameworkInfo {
	info := frameworkInfo{}
	s := string(data)
	if m := gradleBootVersion.FindStringSubmatch(s); m != nil {
		info.framework, info.version = "spring-boot", m[1]
	} else if m := gradleBootVariable.FindStringSubmatch(s); m != nil {
		info.framework, info.version = "spring-boot", m[1]
	}
	if gradleSpringdoc.MatchString(s) {
		info.springdoc = true
	}
	return info
}

// findSpecFiles lists the walked files that look like committed
// OpenAPI/Swagger documents (openapi*/swagger* with a yaml/yml/json
// extension), in input order.
func findSpecFiles(files []string) []string {
	var out []string
	for _, f := range files {
		base := strings.ToLower(path.Base(f))
		ext := path.Ext(base)
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}
		stem := strings.TrimSuffix(base, ext)
		if strings.HasPrefix(stem, "openapi") || strings.HasPrefix(stem, "swagger") {
			out = append(out, f)
		}
	}
	return out
}
