# Introduction

[![CI](https://github.com/abhinav/stitchmd/actions/workflows/ci.yml/badge.svg)](https://github.com/abhinav/stitchmd/actions/workflows/ci.yml)

stitchmd is a tool that stitches together several Markdown files
into one large Markdown file.
It aims to make it easier to maintain large, ungainly Markdown files
while still reaping the benefits of a single document where appropriate.

With stitchmd, you pass in a Markdown file (the *summary file*)
that defines a list of references to other Markdown files
and get back a file with the combined contents of all specified files.
See [Usage](usage.md) for a demonstration.

## Features

- **Cross-linking**:
  stitchmd recognizes cross-links between input Markdown files
  and automatically rewrites them into header links in the generated file.
  This keeps your input files, as well as the output file
  independently browsable on websites like GitHub.
- **Header offsetting**:
  stitchmd will adjust heading levels of included Markdown files
  based on the hierarchy you specify in the summary file.
