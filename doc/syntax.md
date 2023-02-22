# Syntax Reference

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

## Page titles

The page title is determined by the following, in-order:

- If the page has a single level 1 heading,
  that's the title for that page.
- Otherwise, the link text specified in the table of contents
  is the title for that page.

<!-- TODO: explain more -->
