package goldast

import "github.com/yuin/goldmark/ast"

// Visitor visits individual nodes in a Goldmark AST.
type Visitor func(ast.Node) error

// Walk is a simpler variant of [ast.Walk] with support for position tracking.
// It does not support enter/exit tracking.
//
// To skip children, return [ErrSkip].
// All other errors will stop the walker.
func Walk(node ast.Node, fn Visitor) error {
	return walk(node, fn)
}

func walk(n ast.Node, visit Visitor) error {
	if err := visit(n); err != nil {
		return err
	}

	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if err := walk(c, visit); err != nil {
			return err
		}
	}

	return nil
}
