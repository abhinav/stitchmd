package goldast

import "github.com/yuin/goldmark/ast"

// CombineAdjacentTexts combines adjacent [ast.Text] children
// of the given node into a single [ast.Text] node.
func CombineAdjacentTexts(n ast.Node, src []byte) {
	if n.ChildCount() <= 1 {
		return // nothing to do
	}

	var current *ast.Text
	for ch := n.FirstChild(); ch != nil; ch = ch.NextSibling() {
		if ch.Kind() != ast.KindText {
			current = nil
			continue
		}

		next := ch.(*ast.Text)
		if current == nil {
			current = next
			continue
		}

		if current.Merge(next, src) {
			n.RemoveChild(n, next)
			ch = current
		} else {
			current = next
		}
	}
}
