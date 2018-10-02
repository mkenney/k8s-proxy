package error

// Caller defines an interface to runtime caller results.
type Caller interface {
	File() string
	Line() int
	Ok() bool
	Pc() uintptr
	String() string
}
