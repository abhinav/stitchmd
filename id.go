package main

import "strconv"

// idGenerator generates slugs for headings.
// It ensures that each slug is unique.
type idGenerator struct {
	used map[string]struct{}
}

func newIDGenerator() *idGenerator {
	return &idGenerator{
		used: make(map[string]struct{}),
	}
}

// GenerateID generates a unique ID slug for a heading.
// It reports whether a header with this title automatically gets this slug.
// If it returns false, the caller should render an anchor for the slug.
func (g *idGenerator) GenerateID(title string) (slug string, auto bool) {
	slug = titleSlug(title)
	for i := 0; ; i++ {
		slug := slug
		if i > 0 {
			slug = slug + "-" + strconv.Itoa(i)
		}
		if _, ok := g.used[slug]; !ok {
			g.used[slug] = struct{}{}
			return slug, i == 0
		}
	}
}
