# Options

stitchmd supports the following options:

- [`-o FILE`](#write-to-file)
- [`-C DIR`](#change-the-directory)

## Read from stdin

Instead of reading from a specific file on-disk,
you can pass in '-' as the file name to read the summary from stdin.

```bash
cat summary.md | stitchmd -
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
