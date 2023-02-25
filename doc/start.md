# Getting Started

This is a step-by-step tutorial to introduce stitchmd.

For details on how to use it, see [Usage](usage.md).

1. First, [install stitchmd](install.md).
   If you have Go installed, this is as simple as:

     ```bash
     go install go.abhg.dev/stitchmd@latest
     ```

   For other installation methods, see the [Installation](install.md) section.

2. Create a couple Markdown files.
   Feel free to open these up and add content to them.

    ```bash
    echo 'Welcome to my program.' > intro.md
    echo 'It has many features.' > features.md
    echo 'Download it from GitHub.' > install.md
    ```

    Alternatively, clone this repository and copy the [doc folder](./).

3. Create a summary file defining the layout between these files.

    ```bash
    cat > summary.md << EOF
    - [Introduction](intro.md)
      - [Features](features.md)
    - [Installation](install.md)
    EOF
    ```

4. Run stitchmd on the summary.

    ```bash
    stitchmd summary.md
    ```

    The output should look similar to the following:

    ```markdown
    - [Introduction](#introduction)
      - [Features](#features)
    - [Installation](#installation)

    # Introduction

    Welcome to my program.

    ## Features

    It has many features.

    # Installation

    Download it from GitHub.
    ```

    Each included document got its own heading
    matching its level in the summary file.

5. Next, open up `intro.md` and add the following to the bottom:

    ```markdown
    See [installation](install.md) for instructions.
    ```

    If you run stitchmd now, the output should change slightly.

    ```markdown
    - [Introduction](#introduction)
      - [Features](#features)
    - [Installation](#installation)

    # Introduction

    Welcome to my program.
    See [installation](#installation) for instructions.

    ## Features

    It has many features.

    # Installation

    Download it from GitHub.
    ```

    stitchmd recognized the link from `intro.md` to `install.md`,
    and updated it to point to the `# Installation` header instead.

**Next steps**: Play around with the document further:

- Alter the hierarchy further.
- Add an item to the list without a file:

    ```markdown
    - Overview
      - [Introduction](intro.md)
      - [Features](features.md)
    ```

- Add sections or subsections to a document and link to those.

    ```markdown
    [Build from source](install.md#build-from-source).
    ```

- Add a heading to the `summary.md`:

    ```markdown
    # my awesome program

    - [Introduction](#introduction)
      - [Features](#features)
    - [Installation](#installation)
    ```
