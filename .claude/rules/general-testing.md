# General Testing Rules

Minimum **80%** test coverage.

## Test Structure (REQUIRED)
ALWAYS follow the AAA (Arrange, Act, Assert) pattern to structure tests:

- Arrange: Prepare SUT (System Under Test) and dependencies. Use helper functions or factory methods over complex setup for readability.
- Act: Execute the behavior. Ideally single line of code.
- Assert: Verify the result (return value or final state). Use explicit assertions.
- Separation: Visually separate sections with blank lines.

For multiple cases, prefer **Table-Driven Tests** (or Parameterized Tests) with clear test case names.

## Naming Convention
ALWAYS name tests as Facts/Scenarios, not method names:

- Do: `TestSumOfTwoNumbers` or `test_delivery_with_past_date_is_invalid`
- Don't: `TestIsDeliveryValid_InvalidDate_ReturnsFalse` (Implementation details leak)
- Subtests: Use descriptive scenario names for parameterized/table-driven tests.
- Goal: Non-developers should understand the business rule from the name.

## Mocks & Test Doubles
ONLY mock Unmanaged Dependencies (external systems like Email, Message Bus) at the system boundary.

- Managed Dependencies (Database): Do NOT mock. Use real instances (Integration Tests) or in-memory replacements strictly controlled by the test.
- Internal Types: Do NOT mock. Test them as part of the SUT (Classical School approach).
- Interfaces/Protocols: Define at the consumer side, not the implementation side (Dependency Inversion).
- Stubs vs Mocks:
  - Stub (Query): Provides input. Do NOT assert interaction.
  - Mock (Command): Verifies side-effects (e.g., sending email). Assert interaction.

## Refactoring Resistance (CRITICAL)
ALWAYS verify Observable Behavior, NEVER implementation details.

- Black-box testing: Test what the system does, not how.
- Non-public Members: Do NOT test directly. Test through public APIs.
- Fragility: If a refactoring (without behavior change) breaks the test, the test is low quality (False Positive).

## Testing Styles
Prioritize styles in the following order to maximize value:

1. Output-based: Verify return values of pure functions (Highest Maintainability).
2. State-based: Verify state changes of the SUT or collaborators.
3. Communication-based: Verify calls to external systems using Mocks (Use sparingly).