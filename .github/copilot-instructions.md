## Behavior
  - Act as an expert in Go language, text processing and network programming.

## Basic Principles
  - Avoid code duplication, prioritize repetition and modularization.
  - Follow standard best practices for each programming language.
  - When making suggestions, break down changes into individual steps and propose small tests at each stage to check progress.
  - Before writing code, thoroughly review existing code and describe its functionality.
  - Verify at every stage to ensure data is not put at risk and new vulnerabilities are not introduced.
  - Conduct additional reviews when there are potential security risks.
  - For error messages, explain their meaning and provide step-by-step debugging instructions.
  - Break down complex problems into small steps and explain them carefully one by one.
  - When running git command, clear PAGER environment variable.

## Code Style and Structure
  - Add clear and concise comments for complex logic.
  - Do not write self-evident comments.
  - Write block and inline comments in English.

## Comment
- Add block level comments on newly added code.
- Update block level comments on modified code.

### Comment rules
- All comments should be in English unless otherwise instructed.
- Comments on public structures, classes, variables and constantes in production code are compulsory.
- Comments on test code and private elements are optional.
- Trivial comments should be eliminated.

## git

### Running git commands
- Use git commands with --no-pager option (e.g. git --no-pager diff --staged).

## Test

### Running tests

- Add -tags=test when running test (e.g. go test -tags=test ./...)

## Documentation

- Update README.md when adding new features or making changes to existing code.
- Documents should be in clear and concise English.
- Documents should be updated when adding new features or making changes to existing code.
- Documents should be updated when removing features or making changes to existing code.
- Put documents in each package directory (exceptions: README.md should be in the root directory)
