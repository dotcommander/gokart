package main

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"strings"
	"time"
)

// ExistingFilePolicy controls behavior when destination files already exist.
type ExistingFilePolicy string

const (
	ExistingFilePolicyFail      ExistingFilePolicy = "fail"
	ExistingFilePolicySkip      ExistingFilePolicy = "skip"
	ExistingFilePolicyOverwrite ExistingFilePolicy = "overwrite"
)

// ApplyOptions controls scaffolding behavior.
type ApplyOptions struct {
	DryRun             bool
	ExistingFilePolicy ExistingFilePolicy
	SkipManifest       bool
	ManifestMetadata   *scaffoldManifestMetadata
}

// ApplyResult summarizes what the scaffolder did (or would do, in dry-run mode).
type ApplyResult struct {
	Created     []string `json:"created,omitempty"`
	Overwritten []string `json:"overwritten,omitempty"`
	Skipped     []string `json:"skipped,omitempty"`
	Unchanged   []string `json:"unchanged,omitempty"`
}

// ExistingFileConflictError is returned when policy is fail and one or more destination files already exist.
type ExistingFileConflictError struct {
	Paths []string
}

func (e *ExistingFileConflictError) Error() string {
	if len(e.Paths) == 0 {
		return "destination files already exist"
	}

	if len(e.Paths) == 1 {
		return fmt.Sprintf("destination file %s already exists (use --force to overwrite or --skip-existing to keep existing files)", e.Paths[0])
	}

	return fmt.Sprintf("%d destination files already exist (use --force to overwrite or --skip-existing to keep existing files)", len(e.Paths))
}

// ApplyLockError indicates another scaffolding operation already holds the target lock.
type ApplyLockError struct {
	TargetDir string
	Reason    string
	PID       int
	CreatedAt time.Time
}

func (e *ApplyLockError) Error() string {
	if e == nil {
		return "another gokart scaffold is already running"
	}

	var parts []string
	if e.PID > 0 {
		parts = append(parts, fmt.Sprintf("PID %d", e.PID))
	}
	if !e.CreatedAt.IsZero() {
		parts = append(parts, fmt.Sprintf("age %s", time.Since(e.CreatedAt).Truncate(time.Second)))
	}
	if strings.TrimSpace(e.Reason) != "" {
		parts = append(parts, e.Reason)
	}

	detail := strings.Join(parts, ", ")
	if detail == "" {
		return "another gokart scaffold is already running for " + e.TargetDir
	}
	return fmt.Sprintf("another gokart scaffold is already running for %s (%s)", e.TargetDir, detail)
}

type planAction uint8

const (
	planActionCreate planAction = iota
	planActionOverwrite
	planActionSkip
	planActionUnchanged
)

type plannedFile struct {
	RelPath      string
	TargetRoot   string
	DestPath     string
	Rendered     []byte
	Action       planAction
	Existing     []byte
	ExistingMode fs.FileMode
	ExistingInfo fs.FileInfo
	Fingerprint  fileFingerprint
}

type fileFingerprint struct {
	ContentHash [sha256.Size]byte
	Mode        fs.FileMode
	Size        int64
	ModTimeNano int64
}

type rollbackKind uint8

const (
	rollbackRemove rollbackKind = iota
	rollbackRestore
)

type rollbackAction struct {
	Kind           rollbackKind
	Root           string
	Path           string
	Content        []byte
	Mode           fs.FileMode
	ExpectedExists bool
	ExpectedHash   string
	ExpectedSize   int64
	ExpectedMode   fs.FileMode
}

const (
	applyJournalVersion  = 2
	scaffoldManifestPath = ".gokart-manifest.json"
	scaffoldManifestV1   = 1
	scaffoldManifestV2   = 2
	applyLockStaleAfter  = 30 * time.Minute

	modeFlat       = "flat"
	modeStructured = "structured"
)

type applyJournal struct {
	Path  string
	State applyJournalState
}

type applyJournalState struct {
	Version    int                 `json:"version"`
	ID         string              `json:"id"`
	TargetRoot string              `json:"target_root"`
	CreatedAt  time.Time           `json:"created_at"`
	Completed  bool                `json:"completed"`
	Actions    []applyJournalEntry `json:"actions,omitempty"`
}

type applyJournalEntry struct {
	Kind           string `json:"kind"`
	RelPath        string `json:"rel_path"`
	Content        []byte `json:"content,omitempty"`
	Mode           uint32 `json:"mode,omitempty"`
	Applied        bool   `json:"applied,omitempty"`
	ExpectedExists bool   `json:"expected_exists,omitempty"`
	ExpectedSHA256 string `json:"expected_sha256,omitempty"`
	ExpectedSize   int64  `json:"expected_size,omitempty"`
	ExpectedMode   uint32 `json:"expected_mode,omitempty"`
}

type scaffoldManifest struct {
	Version            int                    `json:"version"`
	Generator          string                 `json:"generator"`
	TemplateRoot       string                 `json:"template_root"`
	ExistingFilePolicy ExistingFilePolicy     `json:"existing_file_policy"`
	GeneratedAt        *time.Time             `json:"generated_at,omitempty"`
	Files              []scaffoldManifestFile `json:"files"`
	Integrations       *manifestIntegrations  `json:"integrations,omitempty"`
	Mode               string                 `json:"mode,omitempty"`
	Module             string                 `json:"module,omitempty"`
	UseGlobal          *bool                  `json:"use_global,omitempty"`
}

type scaffoldManifestFile struct {
	Path           string `json:"path"`
	Action         string `json:"action"`
	TemplateSHA256 string `json:"template_sha256"`
	ContentSHA256  string `json:"content_sha256"`
	Mode           uint32 `json:"mode"`
}

type manifestIntegrations struct {
	SQLite   bool `json:"sqlite"`
	Postgres bool `json:"postgres"`
	AI       bool `json:"ai"`
	Redis    bool `json:"redis"`
}

type scaffoldManifestMetadata struct {
	Integrations *manifestIntegrations
	Mode         string
	Module       string
	UseGlobal    *bool
}

type journalRecoveryMismatchError struct {
	RelPath string
	Reason  string
}

func (e *journalRecoveryMismatchError) Error() string {
	if e == nil {
		return ""
	}
	if e.RelPath == "" {
		return e.Reason
	}
	return fmt.Sprintf("%s: %s", e.RelPath, e.Reason)
}

// notRegularFileError describes why a path failed the regular-file assertion.
type notRegularFileError struct {
	Path   string
	Reason string
}

func (e *notRegularFileError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Reason)
}
