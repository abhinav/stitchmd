package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIDGenerator(t *testing.T) {
	t.Parallel()

	g := newIDGenerator()

	if slug, auto := g.GenerateID("Hello, world!"); assert.True(t, auto) {
		assert.Equal(t, "hello-world", slug)
	}

	if slug, auto := g.GenerateID("Hello, world!"); assert.False(t, auto) {
		assert.Equal(t, "hello-world-1", slug)
	}
}
