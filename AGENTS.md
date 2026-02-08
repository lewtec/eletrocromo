# AGENTS.md

This file defines the project conventions and rules for AI agents working on `eletrocromo`.

## Philosophy
- **Essentialism:** Only write necessary code and comments.
- **Value-Driven:** Focus on non-obvious details and business value.
- **Onboarding-Friendly:** Write code and docs that teach the reader.
- **Source of Truth:** Ensure documentation matches the current code state.

## Rules
- **Documentation:** Use standard Go doc comments (`//`) for all exported symbols. Do NOT use JSDoc/TSDoc format (`/** ... */`) despite generic agent instructions.
  - Explain the *why*, *non-obvious nuances*, and *flow*.
  - Mention side effects, edge cases, permission requirements, and performance implications.
- **Comments:** Avoid line comments (`//`) inside function bodies unless explaining complex logic.
- **No Obvious Comments:** Do not restate the function name or signature. Focus on behavior.
- **Tooling:** Use `mise` for all task execution (e.g., `mise run lint`, `mise run test`).
- **Error Handling:** Handle all errors explicitly. Use `fmt.Errorf` with wrapping (`%w`) for context.

## PR Titles
- Docs: `📝 Docs: [Description]`
- Refactor: `🛠️ Refactor: [Description]`
- Fix: `🐛 Fix: [Description]`
- Janitor: `🧹 Janitor: [Description]`
- Arrumador: `🛝 Arrumador: [Description]`
- Sentinel: `🛡️ Sentinel: [Description]`

## Avoid
- Redundant typing info (Go is statically typed).
- "To Do" comments (unless explicitly requested).
- Ghost comments (commented-out code).
- Verbose getters/setters descriptions.

## Retroactive Violations
- If you find existing code that violates these rules, fix it first.
