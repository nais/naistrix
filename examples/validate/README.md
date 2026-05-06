# Composing validators

This example shows how to compose several small, focused validators into a
single `ValidateFunc` using `naistrix.ValidateFuncs`.

Each validator in the chain checks one specific property of the input. Validators run in order and the first failure is reported to the user.
