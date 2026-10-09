// Deterministic content builders for the bench repo (w4-03; test strategy
// §5.1). Everything here is a pure function of (seed, file index): the PRNG
// is splitmix64 keyed by both, the name tables are fixed, and no map is ever
// iterated — same plan in, same bytes out (charter B-6).
package benchgen

import (
	"fmt"
	"strings"
)

// packageCount spreads the tree over enough packages to keep directory
// listings short while staying far below the walker's depth cap.
const packageCount = 25

// domains is the neutral §3.1 vocabulary (library / inventory — no real
// product names, no org-specific identifiers).
var domains = []string{
	"book", "member", "author", "loan", "order", "payment", "shelf", "copy",
	"reservation", "fine", "publisher", "series", "bookmark", "tag", "note",
	"catalog", "review", "readingroom", "export", "inventory",
}

// typeName derives the one class/record name of generated file i. Domain
// word + index keeps names unique, sortable and stable across seeds (the
// seed changes content variety, not the namespace).
func typeName(i int) string {
	d := domains[i%len(domains)]
	return exportWord(d) + fmt.Sprintf("%03d", i/len(domains))
}

// pathOf is the REST base path of controller index i, before variant tweaks.
func pathOf(i int) string {
	return domains[i%len(domains)] + "s" // plural collection path
}

// exportWord turns "readingroom" into "Readingroom" (name-safe identifier).
func exportWord(w string) string {
	if w == "" {
		return w
	}
	return strings.ToUpper(w[:1]) + w[1:]
}

// rng is splitmix64 — a 64-bit PRNG whose output depends only on its state,
// so generation is reproducible across Go versions and machines.
type rng struct{ s uint64 }

func newRng(seed int64, salt uint64) *rng {
	return &rng{s: uint64(seed) ^ salt*0x9E3779B97F4A7C15}
}

func (r *rng) next() uint64 {
	r.s += 0x9E3779B97F4A7C15
	z := r.s
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// intn returns a value in [0, n) with n > 0.
func (r *rng) intn(n int) int {
	return int(r.next() % uint64(n))
}

// pick returns one element of xs (xs must not be empty).
func (r *rng) pick(xs []string) string { return xs[int(r.next()%uint64(len(xs)))] }

// header is the shared file header: machine-readable provenance, then one
// line for a human reader (mirrors the corpus §3.1 header convention).
func header(kind, name, pkg string, i int, seed int64, note string) string {
	return fmt.Sprintf("// benchgen: %s %s (seed %d, package %s)\n// %s\n", kind, name, seed, pkg, note)
}

// controller renders one @RestController file with five endpoints. Variant
// i%6 walks the six rule families: a clean plural versioned collection, then
// one representative shape per family (R1xx path naming, R2xx methods, R3xx
// pagination, R4xx payload, R5xx errors, R6xx versioning). All variants are
// valid Java and parse cleanly — the bench measures scanning, not recovery.
func controller(name, pkg string, i int, seed int64) string {

	dto := name + "Dto"
	svc := name + "Service"
	base := pathOf(i)
	variant := i % 6

	var b strings.Builder
	b.WriteString(header("controller", name, pkg, i, seed, controllerNotes[variant]))
	b.WriteString("package " + pkg + ".web;\n\n")
	b.WriteString("import java.util.List;\n\n")
	b.WriteString("import org.springframework.http.ResponseEntity;\n")
	b.WriteString("import org.springframework.web.bind.annotation.*;\n\n")

	// Path + versioning per family (R6xx-01): variants 0-3 versioned, 4 raw.
	switch variant {
	case 1:
		base = singularPath(base) // R1xx-01 shape
	case 2:
		base = base + "/update" // R1xx-02 shape (verb in path)
	case 4:
		// unversioned path (R6xx-01 shape)
	case 5:
		base = "/api/v1/" + base
	}
	fmt.Fprintf(&b, "@RestController\n@RequestMapping(\"%s\")\n", base)
	fmt.Fprintf(&b, "public class %sController {\n\n", name)
	fmt.Fprintf(&b, "    private final %s %s;\n\n", svc, strings.ToLower(svc))
	fmt.Fprintf(&b, "    public %sController(%s %s) {\n        this.%s = %s;\n    }\n\n",
		name, svc, strings.ToLower(svc), strings.ToLower(svc), strings.ToLower(svc))

	// 1. list — pagination present except the R3xx-negative variant.
	switch variant {
	case 5: // list without pagination parameters (R3xx shape)
		fmt.Fprintf(&b, "    @GetMapping\n    public List<%s> list() {\n        return %s.list();\n    }\n\n", dto, strings.ToLower(svc))
	default:
		fmt.Fprintf(&b, "    @GetMapping\n    public List<%s> list(@RequestParam(defaultValue = \"0\") int page,\n            @RequestParam(defaultValue = \"20\") int size) {\n        return %s.list(page, size);\n    }\n\n",
			dto, strings.ToLower(svc))
	}

	// 2. get by id — variant 3 reads the filter from a GET body (R2xx-01 shape).
	switch variant {
	case 3:
		fmt.Fprintf(&b, "    @GetMapping(\"/search\")\n    public %s search(@RequestBody %s filter) {\n        return %s.find(filter);\n    }\n\n", dto, dto, strings.ToLower(svc))
	default:
		fmt.Fprintf(&b, "    @GetMapping(\"/{id}\")\n    public %s get(@PathVariable(\"id\") long id) {\n        return %s.get(id);\n    }\n\n", dto, strings.ToLower(svc))
	}

	// 3. create — POST with a request body.
	fmt.Fprintf(&b, "    @PostMapping\n    public %s create(@RequestBody %s request) {\n        return %s.save(request);\n    }\n\n", dto, dto, strings.ToLower(svc))

	// 4. update — PUT (or PATCH on the singular variant for variety).
	if variant == 1 {
		fmt.Fprintf(&b, "    @PatchMapping(\"/{id}\")\n    public %s update(@PathVariable(\"id\") long id, @RequestBody %s request) {\n        return %s.save(request);\n    }\n\n", dto, dto, strings.ToLower(svc))
	} else {
		fmt.Fprintf(&b, "    @PutMapping(\"/{id}\")\n    public %s update(@PathVariable(\"id\") long id, @RequestBody %s request) {\n        return %s.save(request);\n    }\n\n", dto, dto, strings.ToLower(svc))
	}

	// 5. delete / error mapping (R5xx shape on variant 5).
	switch variant {
	case 5:
		b.WriteString("    @DeleteMapping(\"/{id}\")\n    public ResponseEntity<String> delete(@PathVariable(\"id\") long id) {\n        try {\n            " + strings.ToLower(svc) + ".delete(id);\n            return ResponseEntity.ok(\"deleted\");\n        } catch (IllegalStateException e) {\n            return ResponseEntity.internalServerError().body(e.getMessage());\n        }\n    }\n")
	default:
		b.WriteString("    @DeleteMapping(\"/{id}\")\n    public ResponseEntity<Void> delete(@PathVariable(\"id\") long id) {\n        " + strings.ToLower(svc) + ".delete(id);\n        return ResponseEntity.noContent().build();\n    }\n")
	}
	b.WriteString("}\n")
	return b.String()
}

var controllerNotes = [6]string{
	"Clean versioned plural collection — the compliant baseline shape.",
	"Singular collection path — exercises the R1xx-01 resource-name family.",
	"Verb under the resource path — exercises the R1xx-02 family.",
	"GET with a request body — exercises the R2xx-01 family.",
	"Unversioned base path — exercises the R6xx-01 family.",
	"List without pagination + string error mapping — R3xx/R5xx shapes.",
}

func singularPath(p string) string {
	if strings.HasSuffix(p, "s") {
		return p[:len(p)-1]
	}
	return p
}

// service renders one @Service class: constructor injection, a list/get/save/
// delete quartet, no annotations beyond @Service and @Transactional.
func service(name, pkg string, i int, seed int64) string {
	dto := name + "Dto"
	r := newRng(seed, uint64(i)*3+1)
	impl := r.pick([]string{"InMemory", "Jpa", "Cached"})
	field := strings.ToLower(name)

	var b strings.Builder
	b.WriteString(header("service", name, pkg, i, seed, "Business service behind the matching controller."))
	b.WriteString("package " + pkg + ".service;\n\n")
	b.WriteString("import java.util.List;\nimport java.util.Optional;\n\n")
	b.WriteString("import org.springframework.stereotype.Service;\n")
	b.WriteString("import org.springframework.transaction.annotation.Transactional;\n\n")
	b.WriteString("import " + pkg + ".model." + name + ";\n\n")
	fmt.Fprintf(&b, "@Service\npublic class %sService {\n\n", name)
	fmt.Fprintf(&b, "    private final %s%sRepository repository;\n\n", impl, exportWord(field))
	fmt.Fprintf(&b, "    public %sService(%s%sRepository repository) {\n        this.repository = repository;\n    }\n\n", name, impl, exportWord(field))
	fmt.Fprintf(&b, "    @Transactional(readOnly = true)\n    public List<%s> list() {\n        return repository.findAll();\n    }\n\n", name)
	fmt.Fprintf(&b, "    @Transactional(readOnly = true)\n    public %s get(long id) {\n        return repository.findById(id).orElseThrow();\n    }\n\n", name)
	fmt.Fprintf(&b, "    @Transactional\n    public %s save(%s entity) {\n        return repository.save(entity);\n    }\n\n", name, name)
	fmt.Fprintf(&b, "    @Transactional\n    public void delete(long id) {\n        repository.deleteById(id);\n    }\n\n")
	fmt.Fprintf(&b, "    public Optional<%s> find(%s filter) {\n        return repository.findAll().stream().filter(e -> e.equals(filter)).findFirst();\n    }\n", name, dto)
	b.WriteString("}\n")
	return b.String()
}

// model renders the DTO/entity layer: records and POJOs, one of four shapes —
// clean camelCase record, string-timestamped record (R4xx-02 shape),
// snake-cased POJO (R4xx naming shape) and a clean POJO with getters.
func model(name, pkg string, i int, seed int64) string {
	var b strings.Builder
	b.WriteString(header("model", name, pkg, i, seed, "DTO/entity type referenced by the matching controller."))
	b.WriteString("package " + pkg + ".model;\n\n")
	switch i % 4 {
	case 0:
		fmt.Fprintf(&b, "public record %s(\n        long id,\n        String name,\n        java.time.Instant createdAt) {\n}\n", name)
	case 1:
		fmt.Fprintf(&b, "public record %s(\n        long id,\n        String name,\n        String createdAt) {\n}\n", name)
	case 2:
		fmt.Fprintf(&b, "public class %s {\n\n    private long id;\n    private String display_name;\n    private java.time.Instant createdAt;\n\n    public long getId() {\n        return id;\n    }\n\n    public void setId(long id) {\n        this.id = id;\n    }\n\n    public String getDisplayName() {\n        return display_name;\n    }\n\n    public void setDisplayName(String display_name) {\n        this.display_name = display_name;\n    }\n\n    public java.time.Instant getCreatedAt() {\n        return createdAt;\n    }\n\n    public void setCreatedAt(java.time.Instant createdAt) {\n        this.createdAt = createdAt;\n    }\n}\n", name)
	default:
		fmt.Fprintf(&b, "public class %s {\n\n    private long id;\n    private String name;\n    private java.time.Instant createdAt;\n\n    public long getId() {\n        return id;\n    }\n\n    public String getName() {\n        return name;\n    }\n\n    public java.time.Instant getCreatedAt() {\n        return createdAt;\n    }\n}\n", name)
	}
	return b.String()
}

// support renders non-API code (§5.1: "file nhiễu phi-API — không file rác
// vô nghĩa"): a util class, a constants interface, a domain exception or a
// plain value holder, rotating by index. None carry Spring annotations.
func support(name, pkg string, i int, seed int64) string {
	var b strings.Builder
	b.WriteString(header("support", name, pkg, i, seed, "Non-API support code — must never reach the API surface."))
	b.WriteString("package " + pkg + ".support;\n\n")
	switch i % 4 {
	case 0:
		fmt.Fprintf(&b, "public final class %sUtil {\n\n    private %sUtil() {\n    }\n\n    public static String normalize(String raw) {\n        return raw == null ? \"\" : raw.strip();\n    }\n\n    public static int clamp(int value, int min, int max) {\n        return Math.max(min, Math.min(max, value));\n    }\n}\n", name, name)
	case 1:
		fmt.Fprintf(&b, "public interface %sDefaults {\n\n    int PAGE_SIZE = 20;\n    String SORT_ORDER = \"asc\";\n\n    static String describe() {\n        return \"page size \" + PAGE_SIZE + \" ordered \" + SORT_ORDER;\n    }\n}\n", name)
	case 2:
		fmt.Fprintf(&b, "public class %sException extends RuntimeException {\n\n    public %sException(String message) {\n        super(message);\n    }\n\n    public %sException(String message, Throwable cause) {\n        super(message, cause);\n    }\n}\n", name, name, name)
	default:
		fmt.Fprintf(&b, "public record %sSummary(long count, String label) {\n\n    public %sSummary {\n        if (count < 0) {\n            throw new IllegalArgumentException(\"count must be >= 0\");\n        }\n    }\n}\n", name, name)
	}
	return b.String()
}

// pomXML is the single build file of the generated tree: it pins the Spring
// Boot version so framework detection reports a real value, exactly like the
// golden corpus (§5.2 reports it as part of the environment).
func pomXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>

  <parent>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-parent</artifactId>
    <version>3.2.4</version>
    <relativePath/>
  </parent>

  <groupId>bench</groupId>
  <artifactId>vanguard-bench</artifactId>
  <version>0.0.1-bench</version>
  <name>vanguard-bench</name>
  <description>Deterministic bench repository generated by tools/gen-bench-repo</description>

  <properties>
    <java.version>17</java.version>
  </properties>

  <dependencies>
    <dependency>
      <groupId>org.springframework.boot</groupId>
      <artifactId>spring-boot-starter-web</artifactId>
    </dependency>
  </dependencies>
</project>
`
}
