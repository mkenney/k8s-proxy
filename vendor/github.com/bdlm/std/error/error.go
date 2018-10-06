package error

// Error defines a slightly more robust error interface than the standard
// library.
type Error interface {
	// Code returns the associated error code. A value of 0 should be
	// considered uncoded.
	Code() Code
	// Caller returns the runtime caller information for this error.
	Caller() Caller
	// Err returns the original error
	Err() error
	// Error implements standard library compatibility. Error should return the
	// string associated with the error code, or the error message if there
	// isn't one
	Error() string
	// String is an alias of Error.
	String() string
	// Msg returns the original error message.
	Msg() string
	// Trace returns the full stack trace of this error.
	Trace() Trace
}
