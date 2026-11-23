# Gather input from the user

Gathering input from the user is a common task for CLI tools. naistrix has some helper functions to achieve this located in the `github.com/nais/naistrix/input` package.

Pass the command or hook context to the input helpers so `--non-interactive`
disables prompting even in a terminal. They check cancellation before prompting,
but do not cancel a running prompt.

If you use a third-party prompt library, check `input.IsInteractive(ctx)`
before opening a prompt.

For example, `go run ./examples/input confirm --non-interactive` fails without
asking or confirming, while `go run ./examples/input select-single-option
--non-interactive` still selects its only option without prompting.

## Confirm

If you want to ask the user for confirmation, you can use `Confirm`:

```go
err := app.AddCommand(&naistrix.Command{
    RunFunc: func (ctx context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
        if ok, err := input.Confirm(ctx, "Are you sure you want to continue?"); err != nil {
            return fmt.Errorf("unable to get confirmation: %w", err)
        } else if !ok {
            out.Println("We will not continue.")
            return nil
        }
        
        out.Println("Continuing operation.")
        return nil
    },
    // ...
})
```

## Input

If you want to ask the user for input, you can use `Input`:

```go
err := app.AddCommand(&naistrix.Command{
    RunFunc: func(ctx context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
        name, err := input.Input(ctx, "What is your name?")
        if err != nil {
            return fmt.Errorf("unable to get input: %w", err)
        }
    
        out.Printf("Hello, %s!\n", name)
        return nil
    },
    // ...
})
```

## Secret

Use `Secret` to collect passwords, API tokens, or other sensitive text with typed characters displayed as `*` instead of the actual input:

```go
token, err := input.Secret(ctx, "API token")
if err != nil {
    return fmt.Errorf("unable to get secret: %w", err)
}
// Use token without printing or logging it.
```

Masking hides the characters, not the input's length. The returned text is not trimmed; an empty submission returns an empty string. Do not print or log the returned secret.

Like the other input helpers that prompt, `Secret` requires interactive
standard input and output and an invocation that allows interaction; otherwise it
returns `input.ErrNotInteractive`.

Try it with `go run ./examples/input secret` from the repository root.

## Select

If you have a list of items that you want the user to select from, you can use `Select`:

```go
err := app.AddCommand(&naistrix.Command{
    RunFunc: func(ctx context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
        option, err := input.Select(ctx, "Select an option", []string{"option1", "option2", "option3"})
        if err != nil {
            return fmt.Errorf("unable to get selection: %w", err)
        }

        out.Println("You selected:", option)
        return nil
    },
    // ...
})
```

The slice you pass to `Select` can be of any type, but for complex types it is recommended to implement the `fmt.Stringer` interface to provide a user-friendly representation of the options.
