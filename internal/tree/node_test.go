package tree

import (
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransform(t *testing.T) {
	t.Parallel()

	give := &Node[int]{
		Value: 1,
		List: List[int]{
			{Value: 2, List: List[int]{{Value: 3}}},
			{Value: 4},
		},
	}

	want := &Node[string]{
		Value: "1",
		List: List[string]{
			{Value: "2", List: List[string]{{Value: "3"}}},
			{Value: "4"},
		},
	}

	nodes := make(map[int]int) // value -> child count
	assert.Equal(t, want, Transform(give, func(c Cursor[int]) string {
		nodes[c.Value()] = c.ChildCount()
		return strconv.Itoa(c.Value())
	}))
	assert.Equal(t, map[int]int{
		1: 2,
		2: 1,
		3: 0,
		4: 0,
	}, nodes)
}

func TestWalk(t *testing.T) {
	t.Parallel()

	give := &Node[int]{
		Value: 1,
		List: List[int]{
			{Value: 2, List: List[int]{{Value: 3}}},
			{Value: 4},
		},
	}

	var got []int
	err := give.Walk(func(v int) error {
		got = append(got, v)
		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, []int{1, 2, 3, 4}, got)
}

func TestWalk_error(t *testing.T) {
	t.Parallel()

	give := &Node[int]{
		Value: 1,
		List: List[int]{
			{Value: 2, List: List[int]{{Value: 3}}},
			{Value: 4},
		},
	}

	var got []int
	giveErr := errors.New("great sadness")
	err := give.Walk(func(v int) error {
		if v == 3 {
			return giveErr
		}
		got = append(got, v)
		return nil
	})

	assert.ErrorIs(t, err, giveErr)
	assert.Equal(t, []int{1, 2}, got)
}

func TestString(t *testing.T) {
	t.Parallel()

	give := &Node[int]{
		Value: 1,
		List: List[int]{
			{Value: 2, List: List[int]{{Value: 3}}},
			{Value: 4},
		},
	}

	assert.Equal(t, "{1 [{2 [{3}]} {4}]}", give.String())
}
