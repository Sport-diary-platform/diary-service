package funcs

import "unsafe"

// S2b converts string to a byte slice without memory allocation.
func S2b(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
