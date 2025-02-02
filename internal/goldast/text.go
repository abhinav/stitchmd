package goldast

import (
	"bytes"
	"io"

	"github.com/yuin/goldmark/ast"
)

// Text returns the text for the [ast.String] and [ast.Text] nodes
// in the tree of the given goldmark AST node.
func Text(src []byte, n ast.Node) []byte {
	var buf bytes.Buffer
	writeNodeText(src, &buf, n)
	return buf.Bytes()
}

func writeNodeText(src []byte, dst io.Writer, n ast.Node) {
	switch n := n.(type) {
	case *ast.Text:
		_, _ = dst.Write(n.Segment.Value(src))
	case *ast.String:
		_, _ = dst.Write(n.Value)
	default:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			writeNodeText(src, dst, c)
		}
	}
}
