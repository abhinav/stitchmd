# Page Titles

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

