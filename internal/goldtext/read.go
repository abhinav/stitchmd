// Package goldtext holds tools for interacting with goldmark's text package.
package goldtext

import (
	"io"

	"github.com/yuin/goldmark/text"
)

// Reader is an io.Reader around a Goldmark text.Segments.
type Reader struct {
	// Source is the original source that the segments are for.
	Source []byte

	// Segments is a list of segments of Source
	// that should be exposed by the Reader.
	Segments *text.Segments

	idx int // current segment
	off int // offset in current segment relative to seg.Start
}

var _ io.Reader = (*Reader)(nil)

// Reset resets the position of Reader in its Source and Segments.
func (r *Reader) Reset() {
	r.idx = 0
	r.off = 0
}

// Read reads from the underlying source and segments
// until the given byte slice is filled or the source runs out.
//
// This implements the io.Reader interface.
func (r *Reader) Read(bs []byte) (total int, err error) {
	dst := 0 // position in bs
	for total < len(bs) {
		if r.idx >= r.Segments.Len() {
			// Reached end of all segments.
			return total, io.EOF
		}

		seg := r.Segments.At(r.idx)
		start := seg.Start + r.off
		if start >= seg.Stop {
			// End of this segment.
			// Move to next and try again.
			r.idx++
			r.off = 0
			continue
		}

		n := copy(bs[dst:], r.Source[start:seg.Stop])
		total += n
		r.off += n
		dst += n
	}
	return total, nil
}
