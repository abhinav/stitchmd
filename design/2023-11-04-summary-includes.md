# Distributed summary documents with Summary Includes

**Issue**: [#4](https://github.com/abhinav/stitchmd/issues/4)

Proposes an extension to stitchmd’s summary document format
to support including other partial summary documents.
This enables stitching together of the summary document
from fragments spread across files.

## Background

With stitchmd, the layout of the output is specified in a **summary file**.
The summary file is a Markdown subset:
it’s valid Markdown, but very restricted about what’s allowed.
Individual documents are referenced in an itemized list in the summary file.

Three kinds of items are supported in the list:

* **File reference**:
  References a Markdown file that should be included in the output
  at that position and nesting level.
* **Group**:
  Header under which other items are grouped,
  but does not represent another document.
* **External link**:
  Link to an external resource.
  These are produced verbatim in the output.

**Kinds of items**

```markdown
- [Introduction](intro.md) # ①
- Getting started # ②
  - [Installation](install.md)
  - [Usage](usage.md)
- [Community](https://example.com/discussions) # ③
```

1. File reference
2. Group
3. External link

## Problem

The summary file format is a central place for the output layout.
This presents two obvious problems:

* The summary file can grow large and ungainly
  if you have lots of nested sections in different places.
* If the same information has to be reproduced in different output documents,
  the summary documents for both must duplicate those items
  instead of being able to re-use the shared fragment.

## Design

As a solution, a new kind of summary list item will be added:
**summary include**.
A summary include is a request to load another summary file
and nest its contents at the current position.

### Syntax

The syntax for a summary include item will reuse the Markdown image syntax.

```markdown
![title](file.md)
```

The image syntax includes a general suggestion of embedding something
in the current document so it’s a good fit for this purpose.

### Example

With a summary document like the following:

```markdown
- ![Using the web UI](web/SUMMARY.md)
- ![Using the CLI](cli/SUMMARY.md)
```

stitchmd will load the referenced summaries
and treat the result equivalent to the following:

```markdown
- Using the web UI # ①
  - [Create an account](web/register.md) # ②
  - [Submit a request](web/submit.md)
- Using the CLI
  - [Authorize your terminal](cli/auth.md)
  - [Submit a JSON request](cli/json.md)
```

1. Included summaries are nested under a group named after the link title.
2. Links to files referenced inside included summaries
    are adjusted to be relative to the parent summary file.

## Future work

In the future, we may allow:

* Including another summary file without showing its items in the TOC.
* Body in the summary file below the item list to be reproduced verbatim.

The two pieces combined would fully address the scenario
originally imagined in [#4](https://github.com/abhinav/stitchmd/issues/4).
