# General Coding Rules

## Design Philosophy

This is a **thin wrapper** around llama-server. Do not over-engineer.

### Do

- Keep implementations simple and focused
- Prefer standard library over external dependencies
- Delete unused code completely (no backwards-compatibility hacks)

### Do Not

- Add unnecessary abstractions
- Optimize prematurely
- Design for hypothetical future requirements
- Add features beyond what's requested

## File Organization

- Keep files under 500 lines
- One main type/responsibility per file
- Extract utilities when a file grows too large
