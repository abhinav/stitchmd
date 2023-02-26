# Usage

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

> The format of the summary file is specified in more detail in [Syntax](syntax.md).

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
