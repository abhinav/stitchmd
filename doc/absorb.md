# Absorbing headings

When adding another Markdown file to your summary,
you can pull the headings of the included file in the final output
by adding a YAML front matter block to the top of the file.

```yaml
---
absorb: true
---
```

For example, given the following:

```markdown
<!-- summary.md -->

- [Installation](install.md)
- [Configuration](config.md)

<!-- config.md -->
---
absorb: true
---

# Configuration

## Adding a new user

To add a new user, ...

## Removing a user

To remove a user, ...
```

<details>
<summary>Output</summary>

```markdown
- [Installation](#install)
- [Configuration](#configuration)
  - [Adding a new user](#adding-a-new-user)
  - [Removing a user](#removing-a-user)

# Installation

<!-- ... -->

# Configuration

## Adding a new user

To add a new user, ...

## Removing a user

To remove a user, ...
```

</details>
