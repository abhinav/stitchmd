package goldast

import (
	"errors"
	"fmt"
	"sort"

	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/stitchmd/internal/pos"
)

// ErrorList tracks errors associated with positions of ast.Nodes in a
// document.
type ErrorList struct {
	info interface {
		Position(int) pos.Position
	}
	errs []*posError

	// Reports the position of the given node.
	// Overridden in tests.
	offsetOf func(ast.Node) int
}

// NewErrorList builds an ErrorList
// that uses the provided [pos.Info] to report positions.
func NewErrorList(info *pos.Info) *ErrorList {
	return newErrorList(info, OffsetOf)
}

func newErrorList(
	info interface{ Position(int) pos.Position },
	offsetOf func(ast.Node) int,
) *ErrorList {
	return &ErrorList{info: info, offsetOf: offsetOf}
}

// Pushf builds an error with the given message and arguments,
// and pushes it into the list.
//
// The error is associated with the position of the given node.
func (el *ErrorList) Pushf(n ast.Node, format string, args ...interface{}) {
	el.errs = append(el.errs, &posError{
		Offset: el.offsetOf(n),
		Err:    fmt.Errorf(format, args...),
	})
}

// Len reports the length of the ErrorList.
func (el *ErrorList) Len() int {
	return len(el.errs)
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
