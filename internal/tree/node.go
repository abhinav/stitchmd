// Package tree defines a general purpose tree data structure.
package tree

import "fmt"

// Node implements a general purpose tree data structure.
// Nodes have a value and zero or more children.
type Node[T any] struct {
	Value T
	List  List[T]
}

// Transform creates a new node
// by transforming the given node and all its descendants
// using the given function.
func Transform[T, U any](t *Node[T], f func(T) U) *Node[U] {
	return &Node[U]{
		Value: f(t.Value),
		List:  TransformList(t.List, f),
	}
}

func (n *Node[T]) String() string {
	if len(n.List) == 0 {
		return fmt.Sprintf("{%v}", n.Value)
	}
	return fmt.Sprintf("{%v %v}", n.Value, n.List)
}

// Walk calls the given function on every node and its descendants
// in depth-first order.
// It stops and returns the first error returned by the function.
func (n *Node[T]) Walk(f func(T) error) error {
	if err := f(n.Value); err != nil {
		return err
	}
	return n.List.Walk(f)
}

// List is a collection of nodes.
type List[T any] []*Node[T]

// TransformList creates a new node list
// by transforming the given node list and all its descendants
// using the given function.
func TransformList[T, U any](t List[T], f func(T) U) List[U] {
	if t == nil {
		// Match nilness of the input.
		return nil
	}
	ns := make(List[U], len(t))
	for i, n := range t {
		ns[i] = Transform(n, f)
	}
	return ns
}

// Walk calls the given function on every node and its descendants
// in depth-first order.
// It stops and returns the first error returned by the function.
func (ns *List[T]) Walk(f func(T) error) error {
	for _, n := range *ns {
		if err := n.Walk(f); err != nil {
			return err
		}
	}
	return nil
}
