package pos

import (
	"errors"
	"fmt"
	"sort"
)

// Error wraps an error with position information.
type Error struct {
	Pos Pos
	Err error
}

// Error reports the message from the underlying error.
func (e *Error) Error() string {
	return e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Err
}

// ErrorList is a mutable collection of [Error]s sorted by position.
//
// ErrorList is not thread-safe.
type ErrorList struct {
	conv interface {
		Position(Pos) Position
	}
	errs []*Error
}

// NewErrorList builds an ErrorList
// that uses the provided Converter to report positions.
func NewErrorList(conv *Converter) *ErrorList {
	return newErrorList(conv)
}

func newErrorList(conv interface {
	Position(Pos) Position
},
) *ErrorList {
	return &ErrorList{conv: conv}
}

// Pushf builds an error with the given message and arguments,
// and pushes it into the list.
func (el *ErrorList) Pushf(pos Pos, format string, args ...interface{}) {
	el.errs = append(el.errs, &Error{Pos: pos, Err: fmt.Errorf(format, args...)})
}

// Len reports the length of the ErrorList.
func (el *ErrorList) Len() int {
	return len(el.errs)
}

// Reset clears the list of errors
// so that the ErrorList may be used again.
func (el *ErrorList) Reset() {
	el.errs = el.errs[:0]
}

// Err returns an error that contains all errors in the list
// or nil if the list is empty.
//
// The errors are sorted by position.
func (el *ErrorList) Err() error {
	if len(el.errs) == 0 {
		return nil
	}

	sort.Slice(el.errs, func(i, j int) bool {
		return el.errs[i].Pos < el.errs[j].Pos
	})

	var errs []error
	for _, e := range el.errs {
		errs = append(errs,
			fmt.Errorf("%v:%w", el.conv.Position(e.Pos), e.Err))
	}
	return errors.Join(errs...)
}
