package spring

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The proto declaration reader tests (w3-06): the reader sees service and
// rpc names only — never message bodies (the raw-proto-parser trap the
// brief forbids). Weird files are skipped silently: no diagnostic, no
// error, no finding — the v0.1 contract is "declaration level, best
// effort".

func writeProto(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(rel), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// TestGrpcServicesFromProto — declaration order, several services per
// file, rpc names verbatim, location on the service keyword.
func TestGrpcServicesFromProto(t *testing.T) {
	dir := t.TempDir()
	writeProto(t, dir, "src/main/proto/library.proto", `syntax = "proto3";

package library.v1;

// A whole commented-out service is NOT an API surface:
// service GhostService {
//   rpc GetGhost(GetGhostRequest) returns (Ghost);
// }

service LibraryService {
  rpc GetBook(GetBookRequest) returns (Book);
  rpc SyncData(SyncRequest) returns (SyncResponse);
}

service AuditService {
  rpc ListEvents(ListEventsRequest) returns (ListEventsResponse);
}
`)
	got := grpcServicesFrom(dir, []string{"src/main/proto/library.proto", "src/main/java/A.java"})
	if len(got) != 2 {
		t.Fatalf("services = %d, want 2: %+v", len(got), got)
	}
	if got[0].Name != "LibraryService" || got[1].Name != "AuditService" {
		t.Errorf("names = %q, %q; want LibraryService, AuditService (declaration order)", got[0].Name, got[1].Name)
	}
	if want := []string{"GetBook", "SyncData"}; len(got[0].Rpcs) != 2 || got[0].Rpcs[0] != want[0] || got[0].Rpcs[1] != want[1] {
		t.Errorf("LibraryService rpcs = %v, want %v (commented-out service excluded)", got[0].Rpcs, want)
	}
	if len(got[1].Rpcs) != 1 || got[1].Rpcs[0] != "ListEvents" {
		t.Errorf("AuditService rpcs = %v, want [ListEvents]", got[1].Rpcs)
	}
	if got[0].Location.File != "src/main/proto/library.proto" || got[0].Location.Line == 0 {
		t.Errorf("location = %+v, want the service keyword line", got[0].Location)
	}
}

// TestGrpcServicesSkipsWeirdFiles — the silent-skip contract: unreadable,
// brace-broken and service-keyword-without-body files contribute nothing,
// emit no diagnostics and never abort the walk.
func TestGrpcServicesSkipsWeirdFiles(t *testing.T) {
	dir := t.TempDir()
	writeProto(t, dir, "a/unterminated.proto", "service Broken {\n  rpc GetX(A) returns (B);\n")
	writeProto(t, dir, "b/keywordless.proto", "service ;\n\nservice  {\n rpc NoName(A) returns (B);\n}\n")
	writeProto(t, dir, "c/one-good.proto", "service Good {\n  rpc GetY(A) returns (B);\n}\n")

	got := grpcServicesFrom(dir, []string{"a/unterminated.proto", "b/keywordless.proto", "c/one-good.proto", "missing/ghost.proto"})
	if len(got) != 1 || got[0].Name != "Good" {
		t.Fatalf("services = %+v, want exactly [Good] — weird files skipped silently", got)
	}
}

// TestGrpcServicesSkipsCleanlyWithoutProto — the w3-06 acceptance: a repo
// without .proto files yields no services, no diagnostics, no error.
func TestGrpcServicesSkipsCleanlyWithoutProto(t *testing.T) {
	dir := t.TempDir()
	writeProto(t, dir, "src/Main.java", "class Main {}")
	got := grpcServicesFrom(dir, []string{"src/Main.java"})
	if len(got) != 0 {
		t.Fatalf("services = %+v, want none", got)
	}
}

// TestStripProtoCommentsKeepsLineMap — comment stripping replaces comment
// text with spaces while preserving newlines, so a location taken from the
// stripped text still points at the original line.
func TestStripProtoCommentsKeepsLineMap(t *testing.T) {
	src := "syntax = \"proto3\";\n\n// lead comment\nservice S { /* inline\n block */ rpc GetZ(A) returns (B); }\n"
	out := stripProtoComments(src)
	if got, want := strings.Count(out, "\n"), strings.Count(src, "\n"); got != want {
		t.Fatalf("stripped text changed the line count: %d → %d", want, got)
	}
	if strings.Contains(out, "lead comment") {
		t.Fatalf("line comment survived: %q", out)
	}
	if strings.Contains(out, "block") {
		t.Fatalf("block comment survived: %q", out)
	}
	if !strings.Contains(out, "rpc GetZ(A) returns (B);") {
		t.Fatalf("code inside a comment block was dropped: %q", out)
	}
}
