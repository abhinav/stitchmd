# Usage


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

