# Composing validators

This example shows how to compose several small, focused validators into a
single `ValidateArgsFunc` using `naistrix.ValidateArgsFuncs`, or a single
`ValidateFlagsFunc` using `naistrix.ValidateFlagsFuncs`.

Each validator in the chain checks one specific property of the input. Both helpers run validators
in order, skip nil entries, and return the first error without running the remaining validators.
An empty chain succeeds.

## Flag validators

`ValidateFlagsFuncs` composes callbacks that capture the flag struct. They receive the command context and check synchronized flags before choices and prompts are resolved. They also run during all positional argument completion, including custom handlers and file completion, so keep them side-effect-free. They do not run during flag-value completion, allowing users to complete required flags that are still missing.

The example checks that `--team` is non-empty and contains no spaces or tabs. Run
`go run . register alice --team platform` to provide a team, or omit the flag to use the default team.
Try `--team ''` or `--team 'two teams'` to exercise the validators.

## Argument validators

`ValidateArgsFuncs` composes callbacks that receive both the command context and the complete,
normalized positional arguments. The example checks the username's minimum length, casing, and
whitespace before registering the user. Argument validators do not run during shell completion.
