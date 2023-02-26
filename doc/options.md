# Options

stitchmd supports the following options:

- [`-offset N`](#offset-heading-levels)
- [`-o FILE`](#write-to-file)
- [`-C DIR`](#change-the-directory)

## Read from stdin

Instead of reading from a specific file on-disk,
you can pass in '-' as the file name to read the summary from stdin.

```bash
cat summary.md | stitchmd -
```

## Offset heading levels

stitchmd changes heading levels based on a few factors:

- level of the section heading
- position of the file in the hierarchy of that section
- the file's own title heading

The `-offset` flag allows you to offset all these headings.

For example, the following will push all headings one level down,
so what would normally be level 3 will now be level 4.

```bash
stitchmd -offset 1 summary.md
```

This number may be negative to reduce heading levels.
For example, the following will turn
what would normally be a level 3 heading into a level 2 heading.

```bash
stitchmd -offset -1 summary.md
```

## Write to file

stitchmd writes its output to stdout by default.
Use the `-o` option to write to a file instead.

```bash
stitchmd -o README.md summary.md
```

## Change the directory

Paths in the summary file are considered
**relative to the summary file**.

Use the `-C` flag to change the directory
that stitchmd considers itself to be in.

```bash
stitchmd -C docs summary.md
```

This is especially useful if your summary file is passed via stdin.

```bash
... | stitchmd -C docs -
```

The `-` above tells stitchmd to [read from stdin](#read-from-stdin).
