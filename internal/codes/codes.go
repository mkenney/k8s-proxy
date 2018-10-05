package codes

import (
	errs "github.com/bdlm/errors"
	std "github.com/bdlm/std/error"
)

const (
	// ErrUnspecified - 1000: The error code was unspecified
	ErrUnspecified std.Code = iota + 1000
)

func init() {
	errs.Codes[ErrUnspecified] = errs.ErrCode{Ext: "An unknown error occurred", Int: "An unknown error occurred", HTTP: 500}
}
