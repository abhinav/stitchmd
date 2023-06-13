package errdefer

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClosef(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		var err error
		Closef(&err, nil, "close %q", "foo")
		require.NoError(t, err)
	})

	t.Run("no error", func(t *testing.T) {
		t.Parallel()

		var err error
		Closef(&err, stubCloser{}, "close %q", "foo")
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		var err error
		errFoo := errors.New("foo")
		Closef(&err, stubCloser{Err: errFoo}, "close %q", "foo")
		assert.ErrorIs(t, err, errFoo)
		assert.ErrorContains(t, err, `close "foo"`)
	})

	t.Run("multiple errors", func(t *testing.T) {
		t.Parallel()

		var err error

		errFoo := errors.New("foo")
		Closef(&err, stubCloser{Err: errFoo}, "close %q", "foo")

		errBar := errors.New("bar")
		Closef(&err, stubCloser{Err: errBar}, "close %q", "bar")

		if assert.ErrorIs(t, err, errFoo) {
			assert.ErrorContains(t, err, `close "foo"`)
		}

		if assert.ErrorIs(t, err, errBar) {
			assert.ErrorContains(t, err, `close "bar"`)
		}
	})
}

type stubCloser struct {
	Err error
}

func (c stubCloser) Close() error {
	return c.Err
}
