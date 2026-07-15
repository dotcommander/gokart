# Build a TV guide CLI

Install the tagged CLI:

```bash
go install github.com/dotcommander/gokart/cmd/gokart@v0.13.0
gokart new tvguide --example
cd tvguide
go run . greet --name World
```

You now have a small, verified CLI in one package. This tutorial turns the
generated `greet` example into an offline TV guide using only deterministic
fixture data and the Go standard library.

## 1. Start with the generated example

The flat scaffold keeps everything in `main.go`. Its `run` function accepts an
argument slice and output writers, so command tests do not need to replace
`os.Args`, `os.Stdout`, or `os.Stderr`.

Run the generated test and command before changing them:

```bash
go test ./...
go run . greet --name World
```

## 2. Rename `greet` to `now`

In `main.go`, rename the root field and command type:

```go
type CLI struct {
	Version kong.VersionFlag `name:"version" help:"Print version information and quit."`
	Now     NowCommand       `cmd:"" help:"Show programs airing now."`
}

type NowCommand struct{}
```

The CLI now reads naturally:

```bash
go run . now
```

## 3. Introduce a schedule

Represent each listing with a small value and keep the tutorial data fixed:

```go
type Program struct {
	Channel string
	Starts  string
	Title   string
}

var schedule = []Program{
	{Channel: "WGBH", Starts: "8:00 PM", Title: "Nature: Wild Coast"},
	{Channel: "WGBX", Starts: "8:00 PM", Title: "The Great British Bake Off"},
	{Channel: "WCVB", Starts: "8:30 PM", Title: "Chronicle"},
}
```

Fixture data keeps output and tests repeatable. Loading real listings is a
separate integration decision.

## 4. Render with `text/tabwriter`

Use the standard library to align columns without owning terminal layout code:

```go
func writePrograms(w io.Writer, programs []Program) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "CHANNEL\tSTARTS\tPROGRAM"); err != nil {
		return err
	}
	for _, program := range programs {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\n", program.Channel, program.Starts, program.Title); err != nil {
			return err
		}
	}
	return tw.Flush()
}
```

Return writer errors. A CLI can then report a closed pipe or failed redirected
write instead of silently claiming success.

## 5. Add `--channel`

Add an optional filter to `NowCommand` and keep the selection logic explicit:

```go
type NowCommand struct {
	Channel string `short:"c" help:"Only show this channel."`
}

func (c *NowCommand) Run(kctx *kong.Context) error {
	programs := schedule
	if c.Channel != "" {
		programs = programsForChannel(schedule, c.Channel)
		if len(programs) == 0 {
			return fmt.Errorf("no programs found for channel %q", c.Channel)
		}
	}
	return writePrograms(kctx.Stdout, programs)
}

func programsForChannel(programs []Program, channel string) []Program {
	selected := make([]Program, 0, len(programs))
	for _, program := range programs {
		if strings.EqualFold(program.Channel, channel) {
			selected = append(selected, program)
		}
	}
	return selected
}
```

Remove the generated `Name` and `Loud` fields and the greeting body. Add
`io`, `strings`, and `text/tabwriter` to the existing imports. Keep the generated
`run` and `main` functions unchanged.

The complete implementation is compiled in
[`examples/cli-app/main.go`](../examples/cli-app/main.go). Its command filters
case-insensitively and returns an error when a requested channel has no listing.

```bash
go run . now --channel WGBH
```

Expected output:

```text
CHANNEL  STARTS   PROGRAM
WGBH     8:00 PM  Nature: Wild Coast
```

## 6. Test arguments and output directly

Keep `main` as the process boundary. Replace the generated greeting test with a
command-boundary test:

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunFiltersByChannel(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(t.Context(), []string{"now", "--channel", "WGBH"}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); !strings.Contains(got, "Nature: Wild Coast") || strings.Contains(got, "Chronicle") {
		t.Fatalf("unexpected output:\n%s", got)
	}
}
```

The repository example also covers no-argument usage, invalid commands, full
dispatch, exact filtered output, and writer failures. From a GoKart source
checkout, run it independently:

```bash
cd examples/cli-app
GOWORK=off go test ./...
```

## 7. Build the named binary

Back in the generated `tvguide` directory:

```bash
go test ./...
go build -o tvguide .
./tvguide now
./tvguide now --channel WGBH
```

Flat projects build from `.`. Structured projects use `./cmd`, but this tutorial
stays with the default single-package layout.

## 8. Choose the next input boundary

Keep `Program` and `writePrograms` unchanged while replacing only the fixture
loader.

For a local export, decode a JSON array from a bounded reader:

```go
decoder := json.NewDecoder(io.LimitReader(file, 1<<20))
if err := decoder.Decode(&programs); err != nil {
	return fmt.Errorf("decode schedule: %w", err)
}
```

For an HTTP source, make cancellation part of the request boundary:

```go
req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
if err != nil {
	return fmt.Errorf("create schedule request: %w", err)
}
resp, err := client.Do(req)
```

Provider selection, credentials, licensing, caching, and live listings are
intentionally outside this beginner path. When the CLI needs managed
integrations, start that project with `--structured`; see the
[generator reference](components/generator.md) and the separate
[SQLite CLI tutorial](sqlite-cli.md).
