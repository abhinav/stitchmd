# Including summary files

List items in the following form are requests to include another summary file:

```markdown
- ![title](file.md)
```

The list defined in the included summary file will be nested under this item.
For example, given the following:

```markdown
<!-- use/summary.md -->

- [Installation](install.md)
- [Configuration](config.md)

<!-- maintain/summary.md -->

- [Dashboard](dashboard.md)
- [Troubleshooting](troubleshooting.md)
```

A joint summary file could take the form:

```markdown
- ![Usage](use/summary.md)
- ![Maintenance](maintain/summary.md)
```

<details>
<summary>Output</summary>

```markdown
- [Usage](#usage)
  - [Installation](#installation)
  - [Configuration](#configuration)
- [Maintenance](#maintenance)
  - [Dashboard](#dashboard)
  - [Troubleshooting](#troubleshooting)

# Usage

## Installation

<!-- ... -->

## Configuration

<!-- ... -->

# Maintenance

## Dashboard

<!-- ... -->

## Troubleshooting

<!-- ... -->
```

</details>

Markdown files referenced in the included summary files
are relative to the summary file.
In the example above, the file tree would be:

```
.
|- summary.md
|- use
|  |- summary.md
|  |- install.md
|  '- config.md
'- maintain
   |- summary.md
   |- dashboard.md
   '- troubleshooting.md
```

Limitations of included summary files:

- It is an error to define more than one section
  in an included summary file.
- The section title's heading level does not affect
  the level of items defined in that summary file.
  The position of the included file in the parent file
  determines levelling.
