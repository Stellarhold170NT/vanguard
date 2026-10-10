package spring

import (
	"os"
	"path/filepath"
	"unicode/utf16"

	"github.com/vanguard-lint/vanguard/internal/ir"
)

// The declaration-level .proto reader (w3-06, charter §3.6 R6xx). The
// reader recognizes only `service <Name> { ... }` blocks and the
// `rpc <Name>(...)` declarations inside them — enough for the R6xx-02
// naming check, nothing more (the raw-proto-parser trap is deliberately
// avoided; the brief caps v0.1 at service + rpc names). Anything the
// mini-reader cannot follow (unbalanced braces, a service keyword without
// a body, an unreadable file) is skipped silently: the best-effort
// contract (charter §5.3) and the w3-06 acceptance — a repo without
// .proto files is a clean skip, no diagnostics, no error.

// grpcServicesFrom reads every .proto file of the walked set and returns
// the declared services in file × declaration order.
func grpcServicesFrom(root string, files []string) []ir.GrpcService {
	var out []ir.GrpcService
	for _, f := range files {
		if !hasSuffixFold(f, ".proto") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			continue // unreadable — skip silently
		}
		out = append(out, protoServices(stripProtoComments(string(data)), f)...)
	}
	return out
}

// hasSuffixFold is a case-insensitive suffix test (.PROTO spellings are
// accepted — extensions are case-insensitive on the filesystems we serve).
func hasSuffixFold(s, suffix string) bool {
	return len(s) >= len(suffix) && equalFoldASCII(s[len(s)-len(suffix):], suffix)
}

func equalFoldASCII(a, b string) bool {
	for i := 0; i < len(a); i++ {
		if lowerASCII(a[i]) != lowerASCII(b[i]) {
			return false
		}
	}
	return true
}

func lowerASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 'a' - 'A'
	}
	return b
}

// stripProtoComments replaces // line comments and /* … */ block comments
// (delimiters included) with spaces while preserving every newline and
// every code byte — so offsets into the stripped text still map onto the
// original line/column. String literals are copied verbatim: a "//" inside
// a string literal is data, not a comment.
func stripProtoComments(src string) string {
	var b []byte
	i := 0
	for i < len(src) {
		c := src[i]
		switch {
		case c == '"' || c == '\'':
			// string literal: copy verbatim, honoring escapes
			quote := c
			b = append(b, c)
			i++
			for i < len(src) {
				b = append(b, src[i])
				if src[i] == '\\' && i+1 < len(src) {
					b = append(b, src[i+1])
					i += 2
					continue
				}
				if src[i] == quote {
					i++
					break
				}
				i++
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				b = append(b, ' ')
				i++
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			b = append(b, ' ', ' ')
			i += 2
			for i < len(src) {
				if src[i] == '*' && i+1 < len(src) && src[i+1] == '/' {
					b = append(b, ' ', ' ')
					i += 2
					break
				}
				if src[i] == '\n' {
					b = append(b, '\n')
				} else {
					b = append(b, ' ')
				}
				i++
			}
		default:
			b = append(b, c)
			i++
		}
	}
	return string(b)
}

// isProtoIdentByte reports whether b may appear inside a proto identifier.
func isProtoIdentByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func isProtoIdentStart(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// protoWordAt reports whether the word-boundary-delimited keyword starts
// at i (identifier bytes on both sides disqualify).
func protoWordAt(src string, i int, word string) bool {
	if i+len(word) > len(src) || src[i:i+len(word)] != word {
		return false
	}
	if i > 0 && isProtoIdentByte(src[i-1]) {
		return false
	}
	if j := i + len(word); j < len(src) && isProtoIdentByte(src[j]) {
		return false
	}
	return true
}

// skipSpaces advances past blanks (never across newlines — a declaration
// whose name is not on the same line is not a declaration we accept).
func skipSpaces(src string, i int) int {
	for i < len(src) && (src[i] == ' ' || src[i] == '\t' || src[i] == '\r') {
		i++
	}
	return i
}

// protoIdent reads a proto identifier (letters, digits, underscore, dots
// for qualified names) starting at i; returns it and the offset after it.
func protoIdent(src string, i int) (string, int) {
	start := i
	for i < len(src) && (isProtoIdentByte(src[i]) || src[i] == '.') {
		i++
	}
	if i == start {
		return "", start
	}
	return src[start:i], i
}

// protoServices scans one comment-stripped file for
// `service <Name> { … }` blocks, one GrpcService per block in declaration
// order, the Location on the service keyword.
func protoServices(src, file string) []ir.GrpcService {
	var out []ir.GrpcService
	for i := 0; i < len(src); {
		if !protoWordAt(src, i, "service") {
			i++
			continue
		}
		line, col := lineCol(src, i)
		j := skipSpaces(src, i+len("service"))
		if j >= len(src) || !isProtoIdentStart(src[j]) {
			i += len("service")
			continue // `service` without a name — skip silently
		}
		name, j2 := protoIdent(src, j)
		j = skipSpaces(src, j2)
		if j >= len(src) || src[j] != '{' {
			i += len("service")
			continue // no body on the same line — skip silently
		}
		body, next, ok := protoBraces(src, j)
		if !ok {
			break // unbalanced braces — drop the rest of the file silently
		}
		out = append(out, ir.GrpcService{
			Name:     name,
			Rpcs:     protoRpcs(body),
			Location: ir.Location{File: file, Line: line, Column: col},
		})
		i = next
	}
	return out
}

// protoBraces returns the text between the '{' at i and its matching '}',
// plus the offset just after the closing brace; unbalanced input reports
// !ok.
func protoBraces(src string, i int) (string, int, bool) {
	if i >= len(src) || src[i] != '{' {
		return "", i, false
	}
	depth := 0
	for j := i; j < len(src); j++ {
		switch src[j] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[i+1 : j], j + 1, true
			}
		}
	}
	return "", i, false
}

// protoRpcs lists the rpc names declared in a service body, in declaration
// order. Only the name is read — request/response types and options are
// out of scope for the declaration-level contract.
func protoRpcs(body string) []string {
	var out []string
	for i := 0; i < len(body); {
		if !protoWordAt(body, i, "rpc") {
			i++
			continue
		}
		j := skipSpaces(body, i+len("rpc"))
		if j < len(body) && isProtoIdentStart(body[j]) {
			name, j2 := protoIdent(body, j)
			j = skipSpaces(body, j2)
			if name != "" && j < len(body) && body[j] == '(' {
				out = append(out, name)
				i = j2
				continue
			}
		}
		i += len("rpc")
	}
	return out
}

// lineCol converts a byte offset into the IR's 1-based line and 1-based
// UTF-16 column (a non-BMP rune counts as 2 units, the SARIF columnKind
// convention the IR documents).
func lineCol(src string, offset int) (line, col int) {
	if offset > len(src) {
		offset = len(src)
	}
	line, col = 1, 1
	units := 0
	for i, r := range src {
		if i >= offset {
			break
		}
		if r == '\n' {
			line++
			units = 0
			continue
		}
		units += len(utf16.Encode([]rune{r}))
	}
	return line, units + 1
}
