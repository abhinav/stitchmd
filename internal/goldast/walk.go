package goldast

import (
	"errors"

	"github.com/yuin/goldmark/ast"
)

// ErrSkip is returned by a [Visitor]
// to indicate that the children of the current node should not be visited.
var ErrSkip = errors.New("skip children")

// Visitor visits individual nodes in a Goldmark AST.
type Visitor func(n *Any) error

// Walk is a simpler variant of [ast.Walk] with support for position tracking.
// It does not support enter/exit tracking.
//
// To skip children, return [ErrSkip].
// All other errors will stop the walker.
func Walk[T ast.Node](node *Node[T], fn Visitor) error {
	return walk(node.AsAny(), fn)
}

func walk(n *Any, visit Visitor) error {
	if err := visit(n); err != nil {
		if errors.Is(err, ErrSkip) {
			err = nil
		}
		return err
	}

	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if err := walk(c, visit); err != nil {
			return err
		}
	}

	return nil
}
