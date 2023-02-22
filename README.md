# stitchmd

- [Introduction](#introduction)
- [Installation](#installation)
- [Usage](#usage)
- [Syntax](#syntax-reference)

## Introduction

stitchmd is a tool that stitches together several Markdown files
into one large Markdown file.
It aims to make it easier to maintain large, ungainly Markdown files
while still reaping the benefits of a single document where appropriate.

With stitchmd, you pass in a Markdown file (the *summary file*)
that defines a list of references to other Markdown files
and get back a file with the combined contents of all specified files.
See [Usage](#usage) for a demonstration.

### Features

- **Cross-linking**:
  stitchmd recognizes cross-links between input Markdown files
  and automatically rewrites them into header links in the generated file.
  This keeps your input files, as well as the output file
  independently browsable on websites like GitHub.
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

To use stitchmd, pass in with a Markdown file
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

## Syntax Reference

The **summary file** is the file you pass to stitchmd
to generate your combined Markdown file.

```bash
stitchmd summary.md
```

The summary file is comprised of one or more **sections**.
Each section lists one or more Markdown files,
using nested lists to define a hierarchy.

For example:

```markdown
- [Getting Started](getting-started.md)
    - [Installation](installation.md)
- [Usage](usage.md)
- [API](api.md)
```

If the summary file defines multiple sections,
sections may specify **section headings**:

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

If a section has a heading specified,
headers for files included in that section
will be offset by the level of that section.

In the example above,
because "User Guide" is a level 1 header,
"Getting Started" will be a level 2 header,
and "Installation" will be a level 3 header.

### Page titles

The page title is determined by the following, in-order:

- If the page has a single level 1 heading,
  that's the title for that page.
- Otherwise, the link text specified in the table of contents
  is the title for that page.

<!-- TODO: explain more -->
