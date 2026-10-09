//go:build darwin && cgo

// Package buildconf — darwin cgo link support.
//
// Why this file exists (w6-01): Go's darwin cgo resolver
// ($GOROOT/src/internal/syscall/unix/net_darwin.go) hard-codes
// `//go:cgo_ldflag "-lresolv"` — the flag is applied whenever cgo is
// enabled on darwin, regardless of the netgo build tag — and
// runtime/cgo/cgo.go adds `#cgo darwin,arm64 LDFLAGS: -framework
// CoreFoundation`. zig cc's bundled macOS libc ships neither a libresolv
// stub nor CoreFoundation framework stubs, so cross-compiling the
// tree-sitter cgo grammar for darwin fails at link time with "unable to
// find dynamic system library 'resolv'" / "unable to find framework
// 'CoreFoundation'".
//
// The stubs next to this file are Apple text-based dylib stubs:
//
//   - libresolv.tbd — same install-name as the real dylib
//     (/usr/lib/libresolv.9.dylib) and the three res_9_* symbols the Go
//     resolver imports; at runtime on macOS the binary loads the real
//     system libresolv.9.dylib.
//   - Frameworks/CoreFoundation.framework/CoreFoundation.tbd — empty
//     export list on purpose: the only Go code referencing CoreFoundation
//     (runtime/cgo/gcc_darwin_arm64.c) is inside `#if TARGET_OS_IPHONE`,
//     i.e. dead code on macOS, so the link only needs the framework to
//     resolve; no CF symbol is ever bound.
//
// ${SRCDIR} is expanded by the go tool to the absolute directory of this
// file, so the flags resolve identically in CI, in the sandbox and in
// local builds — no environment-specific paths.

package buildconf

/*
#cgo LDFLAGS: -L${SRCDIR} -F${SRCDIR}/Frameworks
*/
import "C"
