# GoKart CLI API Reference

This document provides a comprehensive API reference for the `gokart/cli` subpackage, which provides CLI application utilities wrapping Cobra and Lipgloss.

## Related Documentation

- [Configuration](/api/gokart#configuration) - Config loading utilities
- [State](/components/state) - CLI state persistence

## Application Builder

### `type App struct`

Builds CLI applications with sensible defaults. Wraps cobra.Command and viper.Viper for configuration management.

---

### `func NewApp(name, version string) *App`

Creates a new CLI application builder.

**Example:**
```go
app := cli.NewApp("myapp", "1.0.0").
    WithDescription("My application").
    WithConfig("config.yaml").
    WithEnvPrefix("MYAPP")

app.AddCommand(serveCmd)
app.AddCommand(migrateCmd)

if err := app.Run(); err != nil {
    os.Exit(1)
}
```

---

### `func (a *App) WithDescription(desc string) *App`

Sets the app description (shown in help text).

---

### `func (a *App) WithLongDescription(long string) *App`

Sets detailed app description.

---

### `func (a *App) WithConfig(path string) *App`

Sets the config file path.

---

### `func (a *App) WithConfigName(name string) *App`

Sets the config file name (without extension) to search for in standard locations.

---

### `func (a *App) WithEnvPrefix(prefix string) *App`

Sets the environment variable prefix.

Automatically binds environment variables and replaces dots/dashes with underscores (e.g., `db.host` → `MYAPP_DB_HOST`).

---

### `func (a *App) WithStandardFlags() *App`

Adds common flags: `--config`, `--verbose`, `--quiet`.

---

### `func (a *App) AddCommand(cmd *cobra.Command) *App`

Adds a subcommand to the application.

---

### `func (a *App) Root() *cobra.Command`

Returns the root cobra command for advanced customization.

---

### `func (a *App) Viper() *viper.Viper`

Returns the viper instance for config access.

---

### `func (a *App) Run() error`

Executes the CLI application.

---

### `func (a *App) RunWithArgs(args []string) error`

Executes with specific arguments (useful for testing).

---

## Command Helpers

### `func Command(use, short string, run func(cmd *cobra.Command, args []string) error) *cobra.Command`

Creates a new cobra command with common setup.

**Example:**
```go
cmd := cli.Command("serve", "Start the server", func(cmd *cobra.Command, args []string) error {
    return server.Run()
})
```

---

### `func CommandWithArgs(use, short string, nArgs int, run func(cmd *cobra.Command, args []string) error) *cobra.Command`

Creates a command that requires exact positional arguments.

---

### `func Group(use, short string) *cobra.Command`

Creates a command group (no run function, just subcommands).

---

## Output Styling

### `func Success(format string, args ...interface{})`

Prints a success message with green checkmark.

---

### `func Error(format string, args ...interface{})`

Prints an error message to stderr with red X.

---

### `func Warning(format string, args ...interface{})`

Prints a warning message with yellow warning symbol.

---

### `func Info(format string, args ...interface{})`

Prints an info message with blue arrow.

---

### `func Dim(format string, args ...interface{})`

Prints dimmed text (gray).

---

### `func Bold(format string, args ...interface{})`

Prints bold text.

---

## Fatal Helpers

### `func Fatal(format string, args ...interface{})`

Prints an error message and exits with code 1.

---

### `func FatalErr(msg string, err error)`

Prints an error message with the error and exits.

---

### `func Must(err error)`

Exits if err is not nil.

---

## Help Styling

### `func SetStyledHelp(cmd *cobra.Command)`

Configures beautiful help output for a command with colored sections.

---

## Editor Bridge

### `func CaptureInput(initial string, extension string) (string, error)`

Opens `$EDITOR` with initial content and returns the edited text.

Useful for capturing long-form input like commit messages, SQL queries, or configuration.

The extension parameter determines the temp file suffix (e.g., "md", "sql", "json"), which helps editors apply syntax highlighting.

Falls back to "vim" if `$EDITOR` is unset.

**Example:**
```go
text, err := cli.CaptureInput("# Enter your notes\n", "md")
if err != nil {
    return err
}
fmt.Println("You entered:", text)
```

---

### `func CaptureInputWithEditor(editor, initial, extension string) (string, error)`

Opens a specific editor with initial content.

Useful for testing or when you want to override the default editor.

**Example:**
```go
// Force nano regardless of $EDITOR
text, err := cli.CaptureInputWithEditor("nano", "", "txt")
```

---

## Spinner

### `type Spinner struct`

Shows an animated spinner with a message for long-running operations.

---

### `func NewSpinner(message string) *Spinner`

Creates a new spinner with a message.

**Example:**
```go
s := cli.NewSpinner("Loading...")
s.Start()
// do work
s.Stop()
```

---

### `func (s *Spinner) WithFrames(frames []string) *Spinner`

Sets custom animation frames.

---

### `func (s *Spinner) WithDelay(d time.Duration) *Spinner`

Sets the animation speed.

---

### `func (s *Spinner) WithWriter(w io.Writer) *Spinner`

Sets the output writer.

---

### `func (s *Spinner) Start()`

Begins the spinner animation.

For long-running operations, prefer `StartWithContext` to ensure cleanup.

---

### `func (s *Spinner) StartWithContext(ctx context.Context)`

Begins the spinner animation with context cancellation.

The spinner stops automatically when the context is cancelled.

---

### `func (s *Spinner) Update(message string)`

Changes the spinner message.

---

### `func (s *Spinner) Stop()`

Stops the spinner and clears the line.

---

### `func (s *Spinner) StopWithMessage(message string)`

Stops and prints a final message.

---

### `func (s *Spinner) StopSuccess(message string)`

Stops and prints a success message.

---

### `func (s *Spinner) StopError(message string)`

Stops and prints an error message.

---

### `func WithSpinner(message string, fn func() error) error`

Runs a function with a spinner, handling success/error.

**Example:**
```go
err := cli.WithSpinner("Processing...", func() error {
    return doSomething()
})
```

---

## Progress

### `type Progress struct`

Shows a simple progress indicator with percentage bar.

---

### `func NewProgress(message string, total int) *Progress`

Creates a progress indicator.

**Example:**
```go
p := cli.NewProgress("Processing files", 100)
for i := 0; i < 100; i++ {
    p.Increment()
    // do work
}
p.Done()
```

---

### `func (p *Progress) SetWriter(w io.Writer) *Progress`

Sets the output writer.

---

### `func (p *Progress) Increment()`

Advances the progress by 1.

---

### `func (p *Progress) Set(current int)`

Sets the current progress value.

---

### `func (p *Progress) Done()`

Completes the progress and moves to a new line.

---

## Tables

### `type Table struct`

Builds styled terminal tables using lipgloss/table.

---

### `func NewTable(headers ...string) *Table`

Creates a new table with headers.

**Example:**
```go
t := cli.NewTable("ID", "Name", "Status")
t.AddRow("1", "Alice", "Active")
t.AddRow("2", "Bob", "Inactive")
t.Print()
```

---

### `func (t *Table) AddRow(values ...string) *Table`

Adds a row to the table.

---

### `func (t *Table) SetWriter(w io.Writer) *Table`

Sets the output writer.

---

### `func (t *Table) Print()`

Renders the table to the configured writer.

---

### `func (t *Table) String() string`

Returns the table as a string.

---

### `func SimpleTable(headers []string, rows [][]string)`

Prints a quick table without building.

**Example:**
```go
cli.SimpleTable(
    []string{"Name", "Value"},
    [][]string{
        {"Host", "localhost"},
        {"Port", "8080"},
    },
)
```

---

### `func KeyValue(data map[string]string)`

Prints a simple key-value list.

**Example:**
```go
cli.KeyValue(map[string]string{
    "Host": "localhost",
    "Port": "8080",
})
```

---

### `func List(items ...string)`

Prints a bulleted list.

**Example:**
```go
cli.List("Item 1", "Item 2", "Item 3")
```

---

### `func NumberedList(items ...string)`

Prints a numbered list.

**Example:**
```go
cli.NumberedList("First", "Second", "Third")
```
