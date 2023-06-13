// Package errdefer provides functions for joining errors
// from defer statements.
package errdefer

import (
	"errors"
	"fmt"
	"io"
)

// Closef closes the given Closer, appending any error to the given error.
// The error is formatted with fmt.Errorf.
//
// Use this with defer with a named return value:
//
//	func foo() (err error) {
//		f, err := os.Open(fname)
//		if err != nil {
//			// ...
//		}
//		defer multierr.Closef(&err, f, "close %q", fname)
func Closef(err *error, f io.Closer, msg string, args ...any) {
	if f == nil {
		return
	}

	if ferr := f.Close(); ferr != nil {
		msg := fmt.Sprintf(msg, args...)
		*err = errors.Join(*err, fmt.Errorf("%s: %w", msg, ferr))
	}
}
