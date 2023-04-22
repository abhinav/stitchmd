# GitHub Action

[stitchmd-action](https://github.com/abhinav/stitchmd-action)
is a GitHub Action that will install and run stitchmd for you in CI.
With stitchmd-action, you can set up GitHub Workflows to:

- Validate that your output file is always up-to-date

  <details>

  ```yaml
  uses: abhinav/stitchmd-action@v1
  with:
    mode: check
    summary: doc/SUMMARY.md
    output: README.md
  ```

  </details>

- Automatically update your output file based on edits

  <details>

  ```yaml
  uses: abhinav/stitchmd-action@v1
  with:
    mode: write
    summary: doc/SUMMARY.md
    output: README.md

  # Optionally, use https://github.com/stefanzweifel/git-auto-commit-action
  # to automatically push these changes.
  ```

  </details>

- Install a binary of stitchmd and implement your own behavior

  <details>

  ```yaml
  uses: abhinav/stitchmd-action@v1
  with:
    mode: install
  ```

  </details>


For more information, see
[stitchmd-action](https://github.com/abhinav/stitchmd-action).
