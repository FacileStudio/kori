package settings

// DerefBool reads a pointer out of a toggle. Every toggle is filled in by
// defaults, so the pointer is never nil by the time it reaches a caller.
func DerefBool(b *bool) bool {
	return b != nil && *b
}

// DerefInt reads a pointer out of an int setting, falling back to zero when a
// layer never mentioned it.
func DerefInt(n *int) int {
	if n == nil {
		return 0
	}
	return *n
}

// DerefInt64 is DerefInt in the width a token count is measured in.
func DerefInt64(n *int64) int64 {
	if n == nil {
		return 0
	}
	return *n
}

// DerefFloat reads a pointer out of a float setting, falling back to zero when
// a layer never mentioned it.
func DerefFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
