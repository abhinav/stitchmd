// Package rawhtml implements improvements on top of Goldmark's native inline
// HTML support.
//
// For context, Goldmark's inline HTML parser breaks apart inline HTML elements
// so that opening and closing tags are treated as separate nodes:
//
//	foo <a href="bar">bar</a> baz
//
// Becomes the following nodes:
//
//	Text{"foo "}
//	RawHTML{"<a href="bar">"}
//	Text{"bar"}
//	RawHTML{"</a>"}
//	Text{" baz"}
//
// This makes it difficult to parse and manipulate inline HTML elements.
//
// To help work around this, rawhtml augments the parser context
// with information about open-close tags.
package rawhtml
