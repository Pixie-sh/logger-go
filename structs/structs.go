package structs

import "unsafe"

// String wrap a normal string into pointer to avoid Gorm unexpected updates
type String = *string

// ToString convert to pointer string
func ToString(normalStr string) String {
	return &normalStr
}

// EmptyString return an empty string
func EmptyString() String {
	return ToString("")
}

// UnsafeString get unsafe string from bytes without allocation
func UnsafeString(str []byte) String {
	return ToString(*(*string)(unsafe.Pointer(&str))) //no allocation
}

// UnsafeBytes returns a byte pointer without allocation.
func UnsafeBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}