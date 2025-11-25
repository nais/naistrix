# Deploy: guide a release and replay it

Build a release plan from a local catalog. Commands print previews; they do not
deploy anything.

![The deploy wizard collects a service and environment, prints a replay command, and runs that command a second time without prompting](example_0.gif)

## Try it

Build the executable so the preview command can be pasted back into the shell:

```sh
demo_dir=$(mktemp -d)
go build -o "$demo_dir/deploy" ./examples/deploy
export PATH="$demo_dir:$PATH"
deploy plan
```

Enter `gateway` and select `staging`. The wizard prints
`deploy plan gateway --environment staging` before printing the plan. Replay
that command or pass `--non-interactive` to require all inputs in CI:

```sh
deploy --non-interactive plan gateway --environment staging
```

`promote` uses automatic argument prompting instead of a wizard. Its release
choices depend on the environment already supplied. Those choices also power
shell completion after installing the script from `deploy completion bash`
(or substituting your shell):

```sh
deploy promote staging
deploy --non-interactive promote staging gateway-v2.1 --notify=false \
  --set 'regions=oslo,bergen' --set 'replicas=2'
deploy check examples/deploy/gateway.yaml
```

## In the code

- [`planCommand`](main.go) collects wizard inputs and then runs the same flag
  and argument validators used for explicit input.
- [`promoteCommand`](main.go) uses the resolved environment to choose a prompt
  and a list of releases. Its `--notify` distinguishes an explicit false from
  an omitted flag, and `--set` preserves commas in values.
- [`checkCommand`](main.go) offers YAML and JSON file completion for a
  repeatable argument and checks the files during execution.
