// +build appengine

package internal

// BytesToString converts byte slice to string.
func BytesToString(b []byte) string {
	return string(b)
}

// StringToBytes converts string to byte slice.
func StringToBytes(s string) []byte {
	return []byte(s)
}
