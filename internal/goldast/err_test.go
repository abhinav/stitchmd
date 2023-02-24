package goldast

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/stitchmd/internal/pos"
)

func TestErrorList(t *testing.T) {
	info := fakeInfo(func(offset int) pos.Position {
		switch offset {
		case 42:
			return pos.Position{File: "foo", Line: 42, Column: 13}
		case 13:
			return pos.Position{File: "foo", Line: 13, Column: 42}
		default:
			t.Errorf("unexpected offset %v", offset)
			panic("unexpected offset")
		}
	})

	doc := ast.NewDocument()
	para := ast.NewParagraph()
	doc.AppendChild(doc, para)

	offsets := map[ast.Node]int{
		doc:  13,
		para: 42,
	}
	offsetOf := func(n ast.Node) int {
		off, ok := offsets[n]
		if !assert.True(t, ok, "unexpected node: %v", n) {
			n.Dump(nil, 0)
		}
		return off
	}

	el := newErrorList(info, offsetOf)

	foo := errors.New("foo")
	bar := errors.New("bar")

	el.Pushf(para, "great sadness: %w", foo)
	el.Pushf(doc, "great joy: %w", bar)

	assert.Equal(t, 2, el.Len())

	err := el.Err()
	assert.Error(t, err)
	assert.Equal(t, "foo:13:42:great joy: bar\nfoo:42:13:great sadness: foo", err.Error())
	assert.ErrorIs(t, err, foo)
	assert.ErrorIs(t, err, bar)
}

type fakeInfo func(int) pos.Position

func (f fakeInfo) Position(offset int) pos.Position {
	return f(offset)
}

func TestPosError(t *testing.T) {
	wrapped := errors.New("great sadness")
	posErr := &posError{Offset: 42, Err: wrapped}

	assert.Equal(t, "great sadness", posErr.Error())
	assert.ErrorIs(t, posErr, wrapped)
}
