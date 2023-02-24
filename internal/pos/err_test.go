package pos

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	wrapped := errors.New("great sadness")
	posErr := &posError{Offset: 42, Err: wrapped}

	assert.Equal(t, "great sadness", posErr.Error())
	assert.ErrorIs(t, posErr, wrapped)
}

func TestErrorList(t *testing.T) {
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

	el := newErrorList(info)

	t.Run("push", func(t *testing.T) {
		defer el.Reset()

		foo := errors.New("foo")
		bar := errors.New("bar")

		el.Pushf(42, "great sadness: %w", foo)
		el.Pushf(13, "great joy: %w", bar)

		assert.Equal(t, 2, el.Len())

		err := el.Err()
		assert.Error(t, err)
		assert.Equal(t, "foo:13:42:great joy: bar\nfoo:42:13:great sadness: foo", err.Error())
		assert.ErrorIs(t, err, foo)
		assert.ErrorIs(t, err, bar)
	})

	t.Run("empty", func(t *testing.T) {
		assert.Equal(t, 0, el.Len())
		assert.NoError(t, el.Err())
	})
}

type fakeInfo func(int) Position

func (f fakeInfo) Position(offset int) Position {
	return f(offset)
}
