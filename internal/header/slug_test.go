package header

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		give string
		want string
	}{
		{"Hello, world!", "hello-world"},
		{"happy 😄 emoji", "happy--emoji"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.give, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Slug(tt.give))
		})
	}
}
