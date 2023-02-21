// Package pos provides position tracking for a file by offset.
package pos

import (
	"fmt"
	"sort"
	"strconv"
)

// Pos is an efficient representation of a position inside a file.
type Pos int

// Position is the human-readable position information
// for a location in a file.
type Position struct {
	File   string // optional
	Line   int
	Column int
}

func (p Position) String() string {
	bs := make([]byte, 0, len(p.File)+7) // 3 digits for line, 2 for column, 2 for ':'
	if len(p.File) > 0 {
		bs = append(bs, []byte(p.File)...)
		bs = append(bs, ':')
	}
	bs = strconv.AppendInt(bs, int64(p.Line), 10)
	bs = append(bs, ':')
	bs = strconv.AppendInt(bs, int64(p.Column), 10)
	return string(bs)
}

// Info holds richer position information.
// It can be used to convert a Pos to a human-readable Position.
type Info struct {
	file string // optional
	size int    // size of file
	// lines is a list of offsets at which each line starts in source.
	//
	// invariant: lines is always non-empty, and lines[0] is always 0.
	lines []int
}

// FromContent builds a position [Info] from the given source data.
//
// Filename is optional.
func FromContent(filename string, src []byte) *Info {
	con := Info{file: filename, size: len(src)}

	var line int // first line starts at 0
	for idx, c := range src {
		if line >= 0 {
			con.lines = append(con.lines, line)
		}
		line = -1
		if c == '\n' {
			line = idx + 1
		}
	}

	return &con
}

// Filename reports the name of the file as passed to FromContent.
func (c *Info) Filename() string {
	return c.file
}

// Position reports the human-readable position
// for the given offset in the file.
//
// Position panics if the offset is out of bounds of the file.
func (c *Info) Position(pos Pos) Position {
	offset := int(pos) // pos is just an offset for now
	if offset == 0 {
		return Position{File: c.file, Line: 1, Column: 1}
	}
	if offset < 0 || c.size <= offset {
		panic(fmt.Sprintf("offset %v is out of bounds [0, %v)", offset, c.size))
	}

	idx := sort.SearchInts(c.lines, offset)
	if idx < len(c.lines) && c.lines[idx] == offset {
		// idx exactly matches the start of that line.
		// Column 1.
		return Position{
			File:   c.file,
			Line:   idx + 1,
			Column: 1,
		}
	}

	// idx is the index at which the offset would be inserted
	// if it was a new line.
	// So if we have lines at [0, 10, 17],
	// offset 15 will hit index 2 (value 17).
	// Its line number is that index (2),
	// and its column number is calculated
	// by subtracting the *previous* index's line start.
	return Position{
		File: c.file,
		Line: idx,
		// Because c.lines[0] is always 0, and offset is always >0,
		// idx is always in (0, len(lines)),
		// and so idx-1 is always valid.
		Column: offset - c.lines[idx-1] + 1,
	}
}
