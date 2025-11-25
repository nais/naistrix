package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/gif"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"golang.org/x/sync/errgroup"
)

// The example name is used as a shell alias in the generated tape.
var exampleNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// tapeData has data for the tape template file.
type tapeData struct {
	Name string

	// WorkDir is the temporary directory relative to the example directory.
	WorkDir string

	// Steps contains the VHS commands for the visible part of the recording.
	Steps string
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(context.Background(), logger); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger) error {
	taskDir := os.Getenv("MISE_TASK_DIR")
	if taskDir == "" {
		return fmt.Errorf("missing MISE_TASK_DIR environment variable")
	}

	projectRoot := os.Getenv("MISE_PROJECT_ROOT")
	if projectRoot == "" {
		return fmt.Errorf("missing MISE_PROJECT_ROOT environment variable")
	}

	repo, err := os.OpenRoot(projectRoot)
	if err != nil {
		return fmt.Errorf("unable to open project directory: %w", err)
	}
	defer func() {
		_ = repo.Close()
	}()

	tapes, err := fs.Glob(repo.FS(), "examples/*/preview.tape")
	if err != nil {
		return fmt.Errorf("unable to find preview tapes: %w", err)
	}
	if len(tapes) == 0 {
		return errors.New("no preview tapes found matching examples/*/preview.tape")
	}

	tapeTemplate, err := template.ParseFiles(filepath.Join(taskDir, "tape.tpl"))
	if err != nil {
		return fmt.Errorf("unable to parse VHS tape template: %w", err)
	}

	eg, egCtx := errgroup.WithContext(ctx)
	for _, tape := range tapes {
		eg.Go(func() error {
			if err := generateExample(egCtx, repo, filepath.Dir(tape), tapeTemplate, log); err != nil {
				return fmt.Errorf("%s: %w", tape, err)
			}
			return nil
		})
	}

	return eg.Wait()
}

// generateExample updates the recording when its terminal output or dimensions change.
func generateExample(ctx context.Context, repo *os.Root, exampleDir string, tapeTemplate *template.Template, log *slog.Logger) error {
	name := filepath.Base(exampleDir)
	if !exampleNamePattern.MatchString(name) {
		return fmt.Errorf("example directory name %q must contain only letters, digits, underscores, or hyphens", name)
	}
	log = log.With("example", name)

	example, err := repo.OpenRoot(exampleDir)
	if err != nil {
		return fmt.Errorf("unable to open example directory: %w", err)
	}
	defer func() {
		_ = example.Close()
	}()

	steps, err := example.ReadFile("preview.tape")
	if err != nil {
		return fmt.Errorf("unable to read recording steps: %w", err)
	}

	workPath, err := os.MkdirTemp(example.Name(), ".vhs-*")
	if err != nil {
		return fmt.Errorf("unable to create temporary recording directory: %w", err)
	}
	workDir := filepath.Base(workPath)
	defer func() {
		if err := example.RemoveAll(workDir); err != nil {
			log.WarnContext(ctx, "unable to remove temporary recording directory", "error", err)
		}
	}()

	data := tapeData{
		Name:    name,
		WorkDir: workDir,
		Steps:   string(steps),
	}

	// #nosec G204 G702 -- The output argument combines a generated directory name and a validated example name; no shell is involved.
	build := exec.CommandContext(ctx, "go", "build", "-o", filepath.Join(workDir, name), ".")
	build.Dir = example.Name()
	if output, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("unable to build example: %w\n%s", err, output)
	}

	var tape bytes.Buffer
	if err := tapeTemplate.Execute(&tape, data); err != nil {
		return fmt.Errorf("unable to execute template: %w", err)
	}

	cmd := exec.CommandContext(ctx, "vhs", "--quiet", "-")
	cmd.Dir = example.Name()
	cmd.Stdin = &tape

	log.InfoContext(ctx, "generating example")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to run VHS: %w\n%s", err, output)
	}

	generatedAscii := filepath.Join(workDir, "recording.ascii")
	newText, err := example.ReadFile(generatedAscii)
	if err != nil {
		return fmt.Errorf("unable to read generated ASCII snapshot: %w", err)
	}
	newText = trimSnapshotPadding(newText)
	if err := example.WriteFile(generatedAscii, newText, 0o600); err != nil {
		return fmt.Errorf("unable to write normalized ASCII snapshot: %w", err)
	}
	generatedGif := filepath.Join(workDir, "recording.gif")
	generatedSize, err := gifDimensions(example, generatedGif)
	if err != nil {
		return fmt.Errorf("unable to read generated GIF: %w\n%s", err, output)
	}

	const asciiPath = "example_0.ascii"
	const gifPath = "example_0.gif"
	oldText, err := example.ReadFile(asciiPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("unable to read existing ASCII snapshot: %w", err)
	}
	if err == nil && bytes.Equal(newText, oldText) {
		if oldSize, err := gifDimensions(example, gifPath); err == nil && oldSize == generatedSize {
			return nil
		}
	}

	// Publish the GIF first so a failed update leaves the old snapshot available for comparison on retry.
	if err := example.Rename(generatedGif, gifPath); err != nil {
		return fmt.Errorf("unable to move generated gif file: %w", err)
	}
	if err := example.Rename(generatedAscii, asciiPath); err != nil {
		return fmt.Errorf("unable to move generated ASCII snapshot: %w", err)
	}

	log.InfoContext(ctx, "updated example recording")
	return nil
}

// VHS separates captured screens with an 80-column rule. The number of empty
// rows below the last visible line can vary while the browser sizes the terminal.
func trimSnapshotPadding(snapshot []byte) []byte {
	separator := []byte("\n" + strings.Repeat("─", 80) + "\n")
	screens := bytes.Split(snapshot, separator)
	for i := range len(screens) - 1 {
		screens[i] = bytes.TrimRight(screens[i], "\n")
	}
	return bytes.Join(screens, separator)
}

func gifDimensions(example *os.Root, path string) ([2]int, error) {
	f, err := example.Open(path)
	if err != nil {
		return [2]int{}, err
	}
	defer func() {
		_ = f.Close()
	}()

	config, err := gif.DecodeConfig(f)
	return [2]int{config.Width, config.Height}, err
}
