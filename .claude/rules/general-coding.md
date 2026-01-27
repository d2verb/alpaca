# General Coding Rules

## Design Philosophy

This is a **thin wrapper** around llama-server. Do not over-engineer.

### Do

- Keep implementations simple and focused
- Prefer standard library over external dependencies
- Delete unused code completely (no backwards-compatibility hacks)
- Return early (guard clauses) to avoid deep nesting
- Add context when propagating errors

### Do Not

- Add unnecessary abstractions
- Optimize prematurely
- Design for hypothetical future requirements
- Ignore errors silently
- Nest deeper than 3 levels

## File Organization

- Keep files under 500 lines
- One main type/responsibility per file
- Extract utilities when a file grows too large