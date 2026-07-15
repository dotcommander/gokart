package generator

import (
	"context"
	"io"
	"os/exec"
	"regexp"
	"time"
)

const (
	defaultPreset          = "cli"
	configScopeAuto        = "auto"
	configScopeLocal       = "local"
	configScopeGlobal      = "global"
	modeFlat               = "flat"
	modeStructured         = "structured"
	integrationNone        = "none"
	templateRootFlat       = "templates/flat"
	templateRootStructured = "templates/structured"
	manifestGenerator      = "gokart"
	manifestActionCreate   = "create"
	appContextPath         = "internal/app/context.go"
	commandsRootPath       = "internal/commands/root.go"
	newFlagFlat            = "flat"
	newFlagStructured      = "structured"
	newFlagModule          = "module"
	newFlagDB              = "db"
	newFlagAI              = "ai"
	newFlagRedis           = "redis"
	newFlagExample         = "example"
	newFlagLocal           = "local"
	newFlagGlobal          = "global"
	newFlagConfigScope     = "config-scope"
	newFlagDryRun          = "dry-run"
	newFlagForce           = "force"
	newFlagSkipExisting    = "skip-existing"
	newFlagNoManifest      = "no-manifest"
	newFlagVerify          = "verify"
	newFlagNoVerify        = "no-verify"
	newFlagVerifyOnly      = "verify-only"
	newFlagVerifyTimeout   = "verify-timeout"
	newFlagJSON            = "json"
	defaultVerifyTimeout   = 5 * time.Minute
)

var (
	projectNamePattern         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	moduleSegmentPattern       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	verifyOnlyIgnoredFlagNames = []string{
		newFlagFlat, newFlagStructured, newFlagModule, newFlagDB, newFlagAI, newFlagRedis, newFlagExample,
		newFlagLocal, newFlagGlobal, newFlagConfigScope, newFlagForce, newFlagSkipExisting, newFlagNoManifest,
	}
)

type ErrorKind string

const (
	ErrorInvalidArguments     ErrorKind = "invalid_arguments"
	ErrorExistingFileConflict ErrorKind = "existing_file_conflict"
	ErrorVerifyFailed         ErrorKind = "verify_failed"
	ErrorTargetLocked         ErrorKind = "target_locked"
	ErrorConfigInitFailed     ErrorKind = "config_init_failed"
	ErrorScaffoldFailed       ErrorKind = "scaffold_failed"
	ErrorManifestNotFound     ErrorKind = "manifest_not_found"
	ErrorFlatModeUnsupported  ErrorKind = "flat_mode_unsupported"
)

type OperationError struct {
	Kind      ErrorKind
	Partial   bool
	Conflicts []string
	Err       error
}

func (e *OperationError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *OperationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type EventKind string

const (
	EventScaffoldStart     EventKind = "scaffold_start"
	EventSubprocessStart   EventKind = "subprocess_start"
	EventVerificationStart EventKind = "verification_start"
)

type Event struct {
	Kind    EventKind
	Message string
}

type Runtime struct {
	Stdout  io.Writer
	Stderr  io.Writer
	Verbose bool
	Report  func(Event)
}

func (r Runtime) report(kind EventKind, message string) {
	if r.Report != nil {
		r.Report(Event{Kind: kind, Message: message})
	}
}

type CommandRunner interface {
	Run(context.Context, Command) error
}

type Command struct {
	Dir    string
	Stdout io.Writer
	Stderr io.Writer
	Name   string
	Args   []string
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, command Command) error {
	cmd := exec.CommandContext(ctx, command.Name, command.Args...)
	cmd.Dir = command.Dir
	cmd.Stdout = command.Stdout
	cmd.Stderr = command.Stderr
	return cmd.Run()
}

type Dependencies struct {
	GeneratorVersion string
	LookupEnv        func(string) (string, bool)
	Runner           CommandRunner
}

type Service struct{ deps Dependencies }

func New(deps Dependencies) *Service {
	if deps.GeneratorVersion == "" {
		deps.GeneratorVersion = "dev"
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = func(string) (string, bool) { return "", false }
	}
	if deps.Runner == nil {
		deps.Runner = execRunner{}
	}
	return &Service{deps: deps}
}

type CreateRequest struct {
	Args          []string
	Flat          bool
	Structured    bool
	Module        string
	DB            string
	AI            bool
	Redis         bool
	Example       bool
	Local         bool
	Global        bool
	ConfigScope   string
	DryRun        bool
	Force         bool
	SkipExisting  bool
	NoManifest    bool
	Verify        bool
	NoVerify      bool
	VerifyOnly    bool
	VerifyTimeout time.Duration
	Changed       map[string]bool
	WorkingDir    string
	lookupEnv     func(string) (string, bool)
}

type CreateResult struct {
	Preset             string
	Mode               string
	ProjectName        string
	TargetDir          string
	Module             string
	ConfigScope        string
	UseGlobal          bool
	DryRun             bool
	WriteManifest      bool
	VerifyRequested    bool
	VerifyOnly         bool
	VerifyRan          bool
	VerifyPassed       bool
	ExistingFilePolicy ExistingFilePolicy
	Warnings           []string
	Conflicts          []string
	Result             *ApplyResult
	NextDir            string
	NextCommand        string
	NextArgs           []string
}

type AddRequest struct {
	Dir           string
	Integrations  []string
	DryRun        bool
	Force         bool
	Verify        bool
	VerifyTimeout time.Duration
}

type AddResult struct {
	Integrations     []string
	Added            []string
	AlreadyPresent   []string
	FilesCreated     []string
	FilesOverwritten []string
	DryRun           bool
	VerifyRequested  bool
	VerifyPassed     bool
	Warnings         []string
}
