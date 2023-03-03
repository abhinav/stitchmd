# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.4.0 - 2023-03-02
### Added
- `-diff` flag as an alias for `-d`.
- `-preface` flag to include contents of another file in the output verbatim.

### Changed
- The output of `-d` is now colored by default if stdout is a terminal.
  Disable with `--color=never`, or by setting `NO_COLOR=1`.

## v0.3.0 - 2023-03-01
### Added
- `-d` flag to print a diff of the output
  instead of writing to the output file.

## v0.2.0 - 2023-02-26
### Added
- `-offset N` flag to offset all headings by a fixed amount
  (positive or negative).
- `-no-toc` flag to stop the table of contents from being rendered
  in the output.

### Changed
- `-o` now creates the output directory if it does not exist.

## v0.1.1 - 2023-02-25
### Fixed
- Fix corner cases with text in the summary getting merged.
- Fix link rewriting on Windows.

## v0.1.0 - 2023-02-24

- Initial release.
