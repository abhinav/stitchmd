package goldast

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuin/goldmark/ast"
)

func TestErrorList(t *testing.T) {
	t.Parallel()

	info := fakeInfo(func(offset int) Position {
		switch offset {
		case 42:
			return Position{File: "foo", Line: 42, Column: 13}
		case 13:
			return Position{File: "foo", Line: 13, Column: 42}
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

	assert.NoError(t, el.Err(), "empty error list")

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

func TestPosError(t *testing.T) {
	t.Parallel()

	wrapped := errors.New("great sadness")
	posErr := &posError{Offset: 42, Err: wrapped}

	assert.Equal(t, "great sadness", posErr.Error())
	assert.ErrorIs(t, posErr, wrapped)
}

type fakeInfo func(int) Position

func (f fakeInfo) Position(offset int) Position {
	return f(offset)
}
