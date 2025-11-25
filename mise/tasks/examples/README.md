# Generate GIFs for examples

Run `mise run examples:generate` from the repository root. Mise supplies the
locked versions of VHS, ttyd, and FFmpeg. VHS also needs a Chromium-compatible
browser and the DejaVu Sans Mono font.

Add `preview.tape` to an example's directory to include it in generation. The
generator discovers `examples/*/preview.tape` automatically. The shared
[`tape.tpl`](./tape.tpl) sets up the terminal and makes the example binary
available as a shell alias named after the directory. The example's tape
contains the visible commands and waits.

Use `Wait+Screen` to check intermediate prompts or results, then `Wait` for the
shell prompt after a command finishes. The template captures only the final
screen as ASCII so that intermediate redraws don't trigger GIF updates.

Each recording writes `example_0.gif` and `example_0.ascii` side by
side in `examples/<name>/`. The generator keeps the existing GIF if the ASCII
snapshot and GIF dimensions match. Keep both files when updating a recording.
