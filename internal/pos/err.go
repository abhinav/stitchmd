package pos

import (
	"errors"
	"fmt"
	"sort"
)

// posError wraps an error with position information.
type posError struct {
	Offset int
	Err    error
}

func (e *posError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *posError) Unwrap() error {
	return e.Err
}

// ErrorList is a mutable collection of [Error]s sorted by position.
//
// ErrorList is not thread-safe.
type ErrorList struct {
	info interface {
		Position(int) Position
	}
	errs []*posError
}

// NewErrorList builds an ErrorList
// that uses the provided [Info] to report positions.
func NewErrorList(info *Info) *ErrorList {
	return newErrorList(info)
}

func newErrorList(info interface{ Position(int) Position }) *ErrorList {
	return &ErrorList{info: info}
}

// Pushf builds an error with the given message and arguments,
// and pushes it into the list.
func (el *ErrorList) Pushf(off int, format string, args ...interface{}) {
	el.errs = append(el.errs, &posError{
		Offset: off,
		Err:    fmt.Errorf(format, args...),
	})
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
		return el.errs[i].Offset < el.errs[j].Offset
	})

	var errs []error
	for _, e := range el.errs {
		errs = append(errs,
			fmt.Errorf("%v:%w", el.info.Position(e.Offset), e.Err))
	}
	return errors.Join(errs...)
}
