# mdreduce

mdreduce reads a Markdown file defining a table of contents
with links to other Markdown files,
and reduces it all to a single Markdown file.

## Usage

To use mdreduce, run it with a Markdown file defining the table of contents
for your combined Markdown file.

```bash
mdreduce toc.md
```

The table of contents in the input file is a list of one or more sections,
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
  If present, they will be rendered as-is.
- Each link in the list must specify a Markdown file.
- List items may be nested to indicate a hierarchy.

The output of mdreduce will be a single Markdown file with the
contents of all the listed files inlined.

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

### Page title

The page title is determined by the following, in-order:

- If the page has a single level 1 heading,
  that's the title for that page.
- Otherwise, the link text specified in the table of contents
  is the title for that page.

## License

This software is made available under the MIT license.
