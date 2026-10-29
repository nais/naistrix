# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for Naistrix. Each
record describes the context, decision, and consequences of a significant design
choice, with API examples and implementation notes where useful.

A proposed ADR is open for discussion, even if implementation work already
exists. An accepted ADR records an agreed design; it does not necessarily mean
the feature has been implemented.

## When to write an ADR

Use an ADR for significant design decisions whose reasoning will help future
contributors. Judge by impact and tradeoffs, not by the number of lines changed.
Examples include:

- New public APIs or substantial changes to existing behavior.
- Architectural changes or new project-wide conventions.
- Dependencies that introduce meaningful constraints or long-term commitments.
- Decisions with competing approaches, non-obvious tradeoffs, or consequences
  that are costly to reverse.

ADRs are usually unnecessary for bug fixes that restore intended behavior,
routine dependency updates, documentation changes, or mechanical refactoring.
Not every small feature or API addition needs its own record.

When in doubt, ask: **Will someone later need to understand why we chose this
approach?** If yes, an ADR is likely useful.

## Decisions

- [ADR-001: Command wizards](001-command-wizards.md) — Interactive command
  workflows, shared flag structs, command previews, and sensitive input handling.

## Adding a decision

Copy [template.md](template.md) to `NNN-short-title.md`, using the next available
number. Start with status `Proposed`, update it as the decision progresses, and
add a link to the list above.
