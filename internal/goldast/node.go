package goldast

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/mdreduce/internal/pos"
)

func posOf(n ast.Node) (pos.Pos, bool) {
	if n == nil {
		return 0, false
	}
	switch n.Type() {
	case ast.TypeDocument:
		return 0, true
	case ast.TypeBlock:
		lines := n.Lines()
		if lines.Len() == 0 {
			return 0, false
		}
		return pos.Pos(lines.At(0).Start), true
	default:
		return 0, false
	}
}

// Node decorates a Goldmark Node with position information.
type Node[T ast.Node] struct {
	Node T

	// Position at which this Node is present in the file.
	// For inline nodes, this is the position of the current block node.
	// This isn't ideal but this is as good as we can get with Goldmark's
	// current tracking information.
	pos pos.Pos

	parent *Node[ast.Node]

	// TODO: Maybe also track source internally.
	// TODO: make Pos private and expose a Pos method?
}

// Wrap wraps a Goldmark AST node to track position information
// during a traversal.
func Wrap[T ast.Node](n T) (*Node[T], error) {
	p, ok := posOf(n)
	if !ok {
		return nil, errors.New("no position information available")
	}

	return &Node[T]{Node: n, pos: p}, nil
}

// Cast casts the value inside a node
// reporting whether the cast was successful.
//
// Typical usage only requires the destination type parameter:
//
//	goldast.Cast[*ast.Link](n)
func Cast[Dst, Src ast.Node](n *Node[Src]) (*Node[Dst], bool) {
	// Note: The Dst type parameter is first
	// so that Src can be omitted in most cases.
	v, ok := any(n.Node).(Dst)
	if !ok {
		return nil, false
	}
	return &Node[Dst]{Node: v, pos: n.pos}, true
}

// MustCast is a variant of [Cast] that panics if the cast fails.
func MustCast[Dst, Src ast.Node](n *Node[Src]) *Node[Dst] {
	v, ok := Cast[Dst](n)
	if !ok {
		var want Dst
		panic(fmt.Sprintf("expected %T, got %T (%v)", want, reflect.TypeOf(n.Node), n.Kind()))
	}
	return v
}

func (n *Node[T]) generalize() *Node[ast.Node] {
	if n == nil {
		return nil
	}
	return &Node[ast.Node]{
		Node:   n.Node,
		pos:    n.pos,
		parent: n.parent,
	}
}

// Pos reports the position of the current block node.
//
// Use [pos.Converter] to convert this into a human-readable format.
func (n *Node[T]) Pos() pos.Pos {
	if n == nil {
		return 0
	}
	return n.pos
}

// Dump dumps a representation of the node AST to stdout.
func (n *Node[T]) Dump(src []byte, depth int) {
	n.Node.Dump(src, depth)
}

// Kind reports the kind of node.
func (n *Node[T]) Kind() ast.NodeKind {
	return n.Node.Kind()
}

// NextSibling returns the next sibling of this node,
// or nil if this is the last node in this chain.
func (n *Node[T]) NextSibling() *Node[ast.Node] {
	return n.relation((ast.Node).NextSibling, n.parent.Pos(), n.parent)
}

// PreviousSibling returns the previous sibling of this node,
// or nil if this is the first node in this chain.
func (n *Node[T]) PreviousSibling() *Node[ast.Node] {
	return n.relation((ast.Node).PreviousSibling, n.parent.Pos(), n.parent)
}

// Parent returns the parent of this node,
// or nil if this is the root node.
func (n *Node[T]) Parent() *Node[ast.Node] {
	return n.parent
}

// ChildCount reports the number of children this node has.
func (n *Node[T]) ChildCount() int {
	if n == nil {
		return 0
	}
	return n.Node.ChildCount()
}

// FirstChild returns the first child of this node,
// or nil if this has no children.
func (n *Node[T]) FirstChild() *Node[ast.Node] {
	return n.relation((ast.Node).FirstChild, n.pos, n.generalize())
}

// LastChild returns the last child of this node,
// or nil if this has no children.
func (n *Node[T]) LastChild() *Node[ast.Node] {
	return n.relation((ast.Node).LastChild, n.pos, n.generalize())
}

func (n *Node[T]) relation(
	relf func(ast.Node) ast.Node,
	fallback pos.Pos,
	parent *Node[ast.Node],
) *Node[ast.Node] {
	if n == nil {
		return nil
	}

	rel := relf(n.Node)
	if rel == nil {
		return nil
	}

	p, ok := posOf(rel)
	if !ok {
		p = fallback
	}

	return &Node[ast.Node]{
		Node:   rel,
		pos:    p,
		parent: parent,
	}
}
