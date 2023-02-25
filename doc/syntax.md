# Syntax

Although the summary file is Markdown,
stitchmd expects it in a very specific format.

The summary file is comprised of one or more **sections**.
Sections have a **section title** specified by a Markdown heading.

<details>
<summary>Example</summary>

```markdown
# Section 1

<!-- contents of section 1 -->

# Section 2

<!-- contents of section 2 -->
```

</details>

If there's only one section, the section title may be omitted.

```
File = Section | (SectionTitle Section)+
```

Each section contains a Markdown list defining one or more **list items**.
List items are one of the following,
and may optionally have another list nested inside them
to indicate a hierarchy.

- **Links** to local Markdown files:
  These files will be included into the output,
  with their contents adjusted to match their place.

    <details>
    <summary>Example</summary>

    ```markdown
    - [Overview](overview.md)
    - [Getting Started](start/install.md)
    ```
    </details>

- **Plain text**:
  These will become standalone headers in the output.
  These **must** have a nested list.

    <details>
    <summary>Example</summary>

    ```markdown
    - Introduction
        - [Overview](overview.md)
        - [Getting Started](start/install.md)
    ```
    </details>


Items listed in a section are rendered together under that section.
A section is rendered in its entirety
before the listing for the next section begins.

<details>
<summary>Example</summary>

**Input**

```markdown
# Section 1

- [Item 1](item-1.md)
- [Item 2](item-2.md)

# Section 2

- [Item 3](item-3.md)
- [Item 4](item-4.md)
```

**Output**

```markdown
# Section 1

- [Item 1](#item-1)
- [Item 2](#item-2)

## Item 1

<!-- ... -->

## Item 2

<!-- ... -->

# Section 2

- [Item 3](#item-3)
- [Item 4](#item-4)

## Item 3

<!-- ... -->

## Item 4

<!-- ... -->
```

</details>

The heading level of a section determines the minimum heading level
for included documents: one plus the section level.

<details>
<summary>Example</summary>

**Input**

```markdown
## User Guide

- [Introduction](intro.md)
```

**Output**

```markdown
## User Guide

- [Introduction](#introduction)

### Introduction

<!-- ... -->
```

</details>
