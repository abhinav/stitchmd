// Package goldast defines wrappers around Goldmark's AST package and types.
// In particular, its [Node] type is able to track position of a block
// in the document that it came from.
package goldast

import "github.com/yuin/goldmark/ast"

// OffsetOf reports the offset of the given node
// in the document that it came from.
// If the node is an inline node, the position of its parent block is returned.
func OffsetOf(n ast.Node) int {
	if n == nil {
		return 0
	}

	for n != nil {
		switch n.Type() {
		case ast.TypeDocument:
			return 0
		case ast.TypeBlock:
			lines := n.Lines()
			if lines.Len() > 0 {
				return lines.At(0).Start
			}
		}

		n = n.Parent()
	}

	return 0
}
