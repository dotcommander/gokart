# Coordinate processes with an advisory file lock

Use `fslock` when two instances of your own program must not enter the same critical section at once — for example, a single-writer state file or a one-time migration.

**Limit first:** advisory locks only work cooperatively. Every process that touches the shared file must acquire the same lock; a process that ignores the lock is not excluded. `fslock` is Unix-only (`ErrUnsupported` elsewhere).

## Install

```bash
go get github.com/dotcommander/gokart/fslock@v0.17.0
```

Tagged with the workspace `v0.17.0` release.

## First action: acquire, work, release

```go
lock, err := fslock.Acquire("/tmp/myapp.lock", true)
if err != nil {
	if errors.Is(err, fslock.ErrLocked) {
		// Another holder exists; decide whether to wait, retry, or exit.
		return err
	}
	return fmt.Errorf("acquire lock: %w", err)
}
defer func() { _ = lock.Unlock() }()

// Critical section: only one process is here at a time.
if err := doExclusiveWork(); err != nil {
	return err
}
return nil
```

Expected observable result: while the first process holds the lock, a second `Acquire` with `nonblocking=true` returns `fslock.ErrLocked`. With `nonblocking=false` it blocks until the holder releases. This closed loop is source-checked against `fslock/lock_test.go` (`TestAcquireUnlock`, `TestNonblockingContention`); it was not executed as a standalone program here.

## How it works

`Acquire(path, nonblocking)` opens or creates the lock file with mode `0600` and takes an exclusive advisory lock on it. The lock is held as long as the returned `*Lock` is open; `Unlock` (or its alias `Close`) releases the lock and closes the file, and is idempotent — calling it twice, including on a nil receiver, returns nil. The lock file itself is never deleted by the library, which is intentional: deleting it would let a third process create and lock a new file while the old one is still held.

## Failure behavior

| Symptom | Cause | Fix |
|---|---|---|
| `fslock.ErrLocked` | another holder, `nonblocking=true` | Retry later, or decide the work is already being done |
| `fslock.ErrUnsupported` | non-Unix platform | This module has no lock support there; choose another coordination mechanism |
| `open lock file: ...` | missing or unwritable directory | Create the directory first, or pick a writable path |

Locking does not make your data safe by itself: keep the critical section short, and read or write the shared file only while holding the lock.

## Next safe variations

- Block instead of failing: pass `nonblocking=false` when waiting is acceptable; it still returns immediately on unsupported platforms.
- Hold the lock only around a file publish: combine with `gokart.AtomicWriteFile` from the [root package](../api/gokart.md) so concurrent readers never see a partial file.

## See also

- [Root configuration and state](../api/gokart.md) — `AtomicWriteFile` and typed state files
- [State](state.md) — small per-user state; use a lock when two processes can write it
