## Prompt
1. If there are unstaged files, display the list and ask for the next step.
  - stop (default)
  - add unstaged files and proceed
  - ignore unstaged files and proceed
  and proceed only after receiving user approval.
2. Run pre-commit hooks and fix any errors.
3. Draft a commit message in English based on the staged changes.
4. Ask for confirmation before committing with the proposed message using a "y/n" prompt, and proceed only with user approval.
5. commit the change and push it to the upstream.

## Rules

### General
- When executing git diff commands, include the --no-pager option. (i.e. git --no-pager diff --staged)

### Commit message
- Commit message must begin with one of the following prefix.
  - fix: Fix bugs.
  - feat: Add new features or files.
  - refactor: no feature changes, improving the code structure, naming, etc.
  - docs: Add or update documentation.
  - revert: Revert previous commit.
  - chore: commit auto generated files, or small changes which don't fall into other categories.
- The commit message should follow this structure:
  ```
  <type>[optional scope]: <short summary>
  [blank line]
  [detailed description]
  [blank line]
  [footer]
  ```
  - Summary line should be concise (max 80 characters).
  - Use imperative mood in the subject line (e.g., "Add feature" not "Added feature").
  - For the detailed description (body):
    - Explain the motivation for the change and contrast with previous behavior.
    - Wrap text at around 72 characters.
    - Use bullet points if necessary.
  - Footer should contain:
    - References to issues or pull requests (e.g., "Fixes #123").
    - Information about breaking changes, prefixed with "BREAKING CHANGE:".
    - Co-author acknowledgments if applicable (e.g., "Co-authored-by: Name <email@example.com>").
