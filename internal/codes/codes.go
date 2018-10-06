package codes

import (
	errs "github.com/bdlm/errors"
	std "github.com/bdlm/std/error"
)

const (
	// Unspecified - 1000: The error code was unspecified.
	Unspecified std.Code = iota + 1000

	// ContextCancelled - The referenced context has been cancelled.
	ContextCancelled
)

func init() {
	errs.Codes[Unspecified] = errs.ErrCode{Ext: "An unknown error occurred", Int: "An unknown error occurred", HTTP: 500}
}
