# Gather input from the user

Gathering input from the user is a common task for CLI tools. naistrix has some helper functions to achieve this.

## Confirm

If you want to ask the user for confirmation, you can use the `Confirm` function:

```go
RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
    if ok, err := input.Confirm("Are you sure you want to continue?"); err != nil {
        return fmt.Errorf("unable to get confirmation: %w", err)
    } else if !ok {
        out.Println("We will not continue.")
        return nil
    }

    out.Println("Continuing operation.")
    return nil
},
```

## Input

If you want to ask the user for input, you can use the `Input` function:

```go
RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
    name, err := input.Input("What is your name?")
    if err != nil {
        return fmt.Errorf("unable to get input: %w", err)
    }

    out.Printf("Hello, %s!\n", name)
    return nil
},
```

## Select

If you have a list of items that you want the user to select from, you can use the `Select` function:

```go
RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
    option, err := input.Select("Select an option", []string{"option1", "option2", "option3"})
    if err != nil {
        return fmt.Errorf("unable to get selection: %w", err)
    }

    out.Println("You selected:", option)
    return nil
},
```

The slice you pass to the `Select` function can be of any type, but for complex types it is recommended to implement the `fmt.Stringer` interface to provide a user-friendly representation of the options.