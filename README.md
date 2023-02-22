# stitchmd

stitchmd is a tool that generates
large Markdown files from several smaller files.
It lets you define your desired layout as a table of contents,
and then it reads and combines all the files together.

See [Usage](#usage) for a demonstration.

## Features

- **Cross-linking**:
  stitchmd recognizes cross-links between input Markdown files
  and automatically rewrites them to be local header links
  in the generated output file.
  This keeps your input files, as well as the output file
  independently browsable.
- **Header offsetting**:
  stitchmd will adjust heading levels of included Markdown files
  based on the hierarchy you specify in the summary file.

## Installation

Install stitchmd from source with the following command:

```bash
$ go install go.abhg.dev/stitchmd@latest
```

<!-- TODO: binary installation once goreleaser is set up. -->

## Usage

To use stitchmd, run it with a Markdown file
defining the layout of your combined file.
This input file is referred to as the **summary file**,
and is typically named "summary.md".

```bash
stitchmd summary.md
```

The table of contents in the summary file is a list of one or more **sections**,
where each section defines an optional title,
and a list of Markdown files specifying a hierarchy for that section.

For example:

```markdown
# User Guide

- [Getting Started](getting-started.md)
    - [Installation](installation.md)
- [Usage](usage.md)
- [API](api.md)

# Appendix

- [How things work](implementation.md)
- [FAQ](faq.md)
```

Some things to note about the input format:

- Section headers are optional.
  If present, they determine the heading levels for the included files.
- Each link in the list must specify a Markdown file.
- List items may be nested to indicate a hierarchy.

<!-- TODO: document syntax explicitly in a separate section. -->

The output of stitchmd will be a single Markdown file with the
contents of all the listed files inline,
with their links rewritten to match their new location.

<details>

For example, the output of the above input file
will be roughly in the following shape:

```markdown
# User Guide

- [Getting Started](#getting-started)
    - [Installation](#installation)
- [Usage](#usage)
- [API](#api)

## Getting Started

<!-- contents of getting-started.md -->

### Installation

<!-- contents of installation.md -->

## Usage

<!-- contents of usage.md -->

## API

<!-- contents of api.md -->

# Appendix

- [How things work](#how-things-work)
- [FAQ](#faq)

## How things work

<!-- contents of implementation.md -->

## FAQ

<!-- contents of faq.md -->
```

</details>

### Page title

The page title is determined by the following, in-order:

- If the page has a single level 1 heading,
  that's the title for that page.
- Otherwise, the link text specified in the table of contents
  is the title for that page.

## License

This software is made available under the MIT license.
