# stitchmd

- [Introduction](#introduction)
  - [Installation](#installation)
- [Usage](#usage)
  - [Syntax](#syntax)
  - [Page Titles](#page-titles)
- [License](#license)

## Introduction

[![CI](https://github.com/abhinav/stitchmd/actions/workflows/ci.yml/badge.svg)](https://github.com/abhinav/stitchmd/actions/workflows/ci.yml)

stitchmd is a tool that stitches together several Markdown files
into one large Markdown file,
making it easier to maintain larger Markdown files.

It lets you define the layout of your final document in a **summary file**,
which it then uses to stitch and interlink other Markdown files with.

![Flow diagram](doc/images/flow.png)

See [Getting Started](doc/start.md) for a tutorial,
or [Usage](#usage) to start using it.

### Features

- **Cross-linking**:
  Recognizes cross-links between files and their headers
  and re-targets them for their new locations.
  This keeps your input and output files
  independently browsable on websites like GitHub.

    <details>
    <summary>Example</summary>

  **Input**

  ```markdown
  [Install](install.md) the program.
  See also, [Overview](#overview).
  ```

  **Output**

  ```markdown
  [Install](#install) the program.
  See also, [Overview](#overview).
  ```

    </details>

- **Relative linking**:
  Rewrites relative images and links to match their new location.

    <details>
    <summary>Example</summary>

  **Input**

  ```markdown
  ![Graph](images/graph.png)
  ```

  **Output**

  ```markdown
  ![Graph](docs/images/graph.png)
  ```

    </details>

- **Header offsetting**:
  Adjusts levels of all headings in included Markdown files
  based on the hierarchy in the summary file.

    <details>
    <summary>Example</summary>

  **Input**

  ```markdown
  - [Introduction](intro.md)
    - [Installation](install.md)
  ```

  **Output**

  ```markdown
  # Introduction

  <!-- contents of intro.md -->

  ## Installation

  <!-- contents of install.md -->
  ```

    </details>

### Use cases

The following is a non-exhaustive list of use cases
where stitchmd may come in handy.

- Maintaining a document with several collaborators
  with reduced risk of merge conflicts.
- Divvying up a document between collaborators by ownership areas.
  Owners will work inside the documents or directories assigned to them.
- Keeping a single-page and multi-page version of the same content.
- Re-using documentation across multiple Markdown documents.
- Preparing initial drafts of long-form content
  from an outline of smaller texts.

...and more.
(Feel free to contribute a PR with your use case.)

### Installation

You can install stitchmd from [pre-built binaries](#binary-installation)
or [from source](#install-from-source).

#### Binary installation

Pre-built binaries of stitchmd are available for different platforms
over a few different mediums.

##### Homebrew

If you use **Homebrew** on macOS or Linux,
run the following command to install stitchmd:

```bash
brew install abhinav/tap/stitchmd
```

##### ArchLinux

If you use **ArchLinux**,
install stitchmd from [AUR](https://aur.archlinux.org/)
using the [stitchmd-bin](https://aur.archlinux.org/packages/stitchmd-bin/)
package.

```bash
git clone https://aur.archlinux.org/stitchmd-bin.git
cd stitchmd-bin
makepkg -si
```

If you use an AUR helper like [yay](https://github.com/Jguer/yay),
run the following command instead:

```go
yay -S stitchmd-bin
```

##### GitHub Releases

For **other platforms**, download a pre-built binary from the
[Releases page](https://github.com/abhinav/stitchmd/releases)
and place it on your `$PATH`.

#### Install from source

To install stitchmd from source, [install Go >= 1.20](https://go.dev/dl/)
and run:

```bash
go install go.abhg.dev/stitchmd@latest
```

## Usage

```
stitchmd [OPTIONS] FILE
```

stitchmd accepts a single Markdown file as input.
This file defines the layout you want in your combined document,
and is referred to as the **summary file**.

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

> The format of the summary file is specified in more detail in [Syntax](#syntax).

Given such a file as input, stitchmd will print a single Markdown file
including the contents of all listed files inline.

<details>
<summary>Example output</summary>

The output of the input file above
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

### Options

stitchmd supports the following options:

- [`-o FILE`](#write-to-file)
- [`-C DIR`](#change-the-directory)

#### Write to file

stitchmd writes its output to stdout by default.
Use the `-o` option to write to a file instead.

```bash
stitchmd -o README.md summary.md
```

#### Change the directory

Paths in the summary file are considered
**relative to the summary file**.

Use the `-C` flag to change the directory
that stitchmd considers itself to be in.

```bash
stitchmd -C docs summary.md
```

This is especially useful if your summary file is passed via stdin.

```bash
... | stitchmd -C docs - # '-' asks it to read from stdin
```

### Syntax

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

### Page Titles

All pages included with stitchmd are assigned a title.

By default, the title is the name of the item in the summary.
For example, given the following:

```markdown
<!-- summary.md -->
- [Introduction](intro.md)

<!-- intro.md -->
Welcome to Foo.
```

The title for `intro.md` is `"Introduction"`.

<details>
<summary>Output</summary>

```markdown
- [Introduction](#introduction)

# Introduction

Welcome to Foo.
```

</details>

A file may specify its own title by adding a heading
that meets the following rules:

- it's a level 1 heading
- it's the first item in the file
- there are no other level 1 headings in the file

If a file specifies its own title,
this does **not** affect its name in the summary list.
This allows the use of short link titles for long headings.

For example, given the following:

```markdown
<!-- summary.md -->
- [Introduction](intro.md)

<!-- intro.md -->
# Introduction to Foo

Welcome to Foo.
```

The title for `intro.md` will be `"Introduction to Foo"`.

<details>
<summary>Output</summary>

```markdown
- [Introduction](#introduction-to-foo)

# Introduction to Foo

Welcome to Foo.
```

</details>

## License

This software is licensed under the MIT license.
