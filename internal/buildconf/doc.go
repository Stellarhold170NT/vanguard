// Package buildconf carries the cgo link-time support files used by the
// release build matrix (w6-01).
//
// It holds no runtime code. Its single purpose is to register #cgo link
// flags for platforms whose C toolchain needs help linking Go's cgo
// dependency set. See darwin_cgo.go for the one case that exists today
// (zig cc + darwin + libresolv).
package buildconf
