package goldast

import (
	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/mdreduce/internal/pos"
)

// Visitor visits individual nodes in a Goldmark AST.
type Visitor func(n *Node[ast.Node], enter bool) (ast.WalkStatus, error)

// Walk is a variant of [ast.Walk] with support for position tracking.
func Walk[T ast.Node](node *Node[T], fn Visitor) error {
	return ast.Walk(node.Node, (&walker{
		posstack: []pos.Pos{node.Pos()},
		visit:    fn,
	}).Visit)
}

type walker struct {
	posstack []pos.Pos // stack
	visit    Visitor
}

func (w *walker) Visit(n ast.Node, enter bool) (ast.WalkStatus, error) {
	pos, ok := posOf(n)
	if !ok {
		pos = w.posstack[len(w.posstack)-1]
	}

	if enter {
		w.posstack = append(w.posstack, pos)
	} else {
		defer func() {
			w.posstack = w.posstack[:len(w.posstack)-1]
		}()
	}

	return w.visit(&Node[ast.Node]{Node: n, pos: pos}, enter)
}
