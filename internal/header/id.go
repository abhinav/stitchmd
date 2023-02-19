package header

import "strconv"

// IDGen generates slugs for headings.
// It ensures that each slug is unique.
type IDGen struct {
	used map[string]struct{}
}

func NewIDGen() *IDGen {
	return &IDGen{
		used: make(map[string]struct{}),
	}
}

// GenerateID generates a unique ID slug for a heading.
// It reports whether a header with this title automatically gets this slug.
// If it returns false, the caller should render an anchor for the slug.
func (g *IDGen) GenerateID(title string) (slug string, auto bool) {
	slug = Slug(title)
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
