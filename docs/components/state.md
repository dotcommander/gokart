# State

Simple state persistence for CLI applications across invocations. Stores typed data as JSON in platform-specific user config directories.

## Installation

```bash
go get github.com/dotcommander/gokart
```

## Quick Start

```go
import "github.com/dotcommander/gokart"

type AppState struct {
    LastOpened  string    `json:"last_opened"`
    WindowSize  int       `json:"window_size"`
    RecentFiles []string  `json:"recent_files"`
}

// Save state
err := gokart.SaveState("myapp", "state.json", AppState{
    LastOpened:  "/path/to/file.txt",
    WindowSize:  1024,
    RecentFiles: []string{"/a.txt", "/b.txt"},
})

// Load state (handle first-run)
state, err := gokart.LoadState[AppState]("myapp", "state.json")
if errors.Is(err, os.ErrNotExist) {
    // First run - use defaults
    state = AppState{WindowSize: 800}
}
```

---

## Platform-Specific Paths

State files are stored in the user config directory:

| Platform | Path Pattern | Example |
|----------|--------------|---------|
| **macOS** | `~/Library/Application Support/{app}/` | `/Users/alice/Library/Application Support/myapp/state.json` |
| **Linux** | `~/.config/{app}/` | `/home/alice/.config/myapp/state.json` |
| **Windows** | `%AppData%\{app}\` | `C:\Users\Alice\AppData\Roaming\myapp\state.json` |

The path is determined by `os.UserConfigDir()`, ensuring files follow platform conventions.

### Getting the State Path

```go
path := gokart.StatePath("myapp", "state.json")
fmt.Printf("State file: %s\n", path)
// macOS: /Users/alice/Library/Application Support/myapp/state.json
// Linux: /home/alice/.config/myapp/state.json
```

Returns empty string if the config directory cannot be determined.

---

## Functions

### SaveState

Saves typed state to the config directory as indented JSON.

```go
func SaveState[T any](appName, filename string, data T) error
```

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `appName` | `string` | Application name (creates subdirectory in config dir) |
| `filename` | `string` | Name of the state file (e.g., `"state.json"`) |
| `data` | `T` | Data to save (any type, typically a struct) |

**Behavior:**

- Creates directory with `0755` permissions if needed
- Writes file with `0644` permissions
- Formats JSON with 2-space indentation for human readability
- Overwrites existing file

**Example:**

```go
type Preferences struct {
    Theme    string `json:"theme"`
    FontSize int    `json:"font_size"`
}

err := gokart.SaveState("myapp", "preferences.json", Preferences{
    Theme:    "dark",
    FontSize: 14,
})
if err != nil {
    log.Fatal(err)
}
```

---

### LoadState

Loads typed state from the config directory.

```go
func LoadState[T any](appName, filename string) (T, error)
```

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `appName` | `string` | Application name |
| `filename` | `string` | Name of the state file |

**Returns:**

- `T`: The loaded data (zero value if error)
- `error`: `os.ErrNotExist` if file doesn't exist, unmarshal error if invalid JSON

**Behavior:**

- Returns zero value and `os.ErrNotExist` if file doesn't exist
- Returns zero value and unmarshal error if JSON is invalid
- Use `errors.Is(err, os.ErrNotExist)` to detect first run

**Example:**

```go
type Preferences struct {
    Theme    string `json:"theme"`
    FontSize int    `json:"font_size"`
}

prefs, err := gokart.LoadState[Preferences]("myapp", "preferences.json")
if errors.Is(err, os.ErrNotExist) {
    // First run - use defaults
    prefs = Preferences{Theme: "light", FontSize: 12}
} else if err != nil {
    log.Fatal(err)
}

fmt.Printf("Theme: %s, Font: %d\n", prefs.Theme, prefs.FontSize)
```

---

### StatePath

Returns the full path to a state file without reading or writing.

```go
func StatePath(appName, filename string) string
```

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `appName` | `string` | Application name |
| `filename` | `string` | Name of the state file |

**Returns:**

- `string`: Full path to state file
- Empty string if config directory cannot be determined

**Example:**

```go
path := gokart.StatePath("myapp", "state.json")
if path == "" {
    log.Fatal("Cannot determine config directory")
}

fmt.Printf("State stored at: %s\n", path)
// macOS: /Users/alice/Library/Application Support/myapp/state.json
// Linux: /home/alice/.config/myapp/state.json
```

---

## Complete Example: Save/Load Cycle

```go
package main

import (
    "errors"
    "fmt"
    "log"
    "os"

    "github.com/dotcommander/gokart"
)

type AppState struct {
    LastFile   string   `json:"last_file"`
    Recent     []string `json:"recent_files"`
    VisitCount int      `json:"visit_count"`
}

func main() {
    // Load existing state or create defaults
    state, err := gokart.LoadState[AppState]("myapp", "state.json")
    if errors.Is(err, os.ErrNotExist) {
        // First run
        state = AppState{
            Recent:     []string{},
            VisitCount: 0,
        }
    } else if err != nil {
        log.Fatal(err)
    }

    // Update state
    state.LastFile = "/new/file.txt"
    state.VisitCount++
    state.Recent = append(state.Recent, "/new/file.txt")
    if len(state.Recent) > 10 {
        state.Recent = state.Recent[1:] // Keep last 10
    }

    // Save updated state
    if err := gokart.SaveState("myapp", "state.json", state); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Visited %d times\n", state.VisitCount)
    fmt.Printf("State saved to: %s\n", gokart.StatePath("myapp", "state.json"))
}
```

---

## State vs Config

| Aspect | State (`SaveState`/`LoadState`) | Config (`LoadConfig`) |
|--------|-------------------------------|----------------------|
| **Purpose** | Runtime data that changes | Application settings |
| **Format** | JSON only | YAML, JSON, TOML, etc |
| **Location** | `~/.config/{app}/` | Project-relative or custom |
| **Human-editable** | Yes (indented JSON) | Yes (structured format) |
| **Use for** | Recent files, last run, counters | DB credentials, API keys, feature flags |

**Use state for:**
- Recent files list
- Last opened document
- Window position/size
- Usage statistics
- Caching expensive computations

**Use config for:**
- Database connection strings
- API credentials
- Feature flags
- Application settings
- Deployment-specific values

---

## File Permissions

State files use standard Unix permissions:

| Resource | Permissions | Meaning |
|----------|-------------|---------|
| **Directory** | `0755` | Owner: rwx, Group/Others: r-x |
| **File** | `0644` | Owner: rw-, Group/Others: r-- |

Files are human-readable (not encrypted) - do not store secrets or sensitive credentials.

---

## Best Practices

### Handle First Run

Always check for `os.ErrNotExist`:

```go
state, err := gokart.LoadState[AppState]("myapp", "state.json")
if errors.Is(err, os.ErrNotExist) {
    // Initialize defaults
    state = AppState{WindowSize: 800}
} else if err != nil {
    return err
}
```

### Use Struct Tags

Label JSON fields for clarity and compatibility:

```go
type State struct {
    LastOpened   string    `json:"last_opened"`   // snake_case in JSON
    Maximized    bool      `json:"maximized"`
    RecentFiles  []string  `json:"recent_files"`
}
```

### Validate on Load

Check loaded data before using:

```go
state, _ := gokart.LoadState[State]("myapp", "state.json")
if state.WindowSize < 400 {
    state.WindowSize = 800 // Corrupt or invalid state
}
```

### Keep State Small

State files are loaded entirely into memory:

```go
// Good - small, simple data
type State struct {
    LastFile  string
    Visited   int
}

// Avoid - large datasets
type State struct {
    Cache     []byte  // DON'T: Use separate cache file
    LogData   string  // DON'T: Use proper logging
}
```

For large data, consider SQLite or a dedicated database.

---

## Thread Safety

`SaveState` and `LoadState` are **not** thread-safe. If accessing from multiple goroutines, use synchronization:

```go
var mu sync.Mutex

mu.Lock()
state, _ := gokart.LoadState[State]("myapp", "state.json")
state.Count++
gokart.SaveState("myapp", "state.json", state)
mu.Unlock()
```

---

## Reference

### Functions

| Function | Returns | Description |
|----------|---------|-------------|
| `SaveState[T any]` | `error` | Save typed state to config directory |
| `LoadState[T any]` | `(T, error)` | Load typed state from config directory |
| `StatePath` | `string` | Get full path to state file |

### See Also

- [`os.UserConfigDir`](https://pkg.go.dev/os#UserConfigDir) - Platform config directory logic
- [`encoding/json`](https://pkg.go.dev/encoding/json) - JSON marshaling/unmarshaling
- [Config](/api/gokart#configuration) - Application configuration
- [Cache](/components/cache) - Server-side caching
