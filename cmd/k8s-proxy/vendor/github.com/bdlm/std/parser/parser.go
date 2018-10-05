package parser

import (
	"io"
)

/*
Parser is the interface that wraps the basic Parse method.

Parser accepts a reader r and parses the data it returns, returning an error
if parsing fails.

Implementations must not retain r. Implementations should not retain any
parsed data if returning an error.
*/
type Parser interface {
	Parse(r io.Reader) (interface{}, error)
}
