# Bug Report: autocd-go ExitWithDirectory() Process Replacement Fails Silently from Root Directory on macOS

## Summary
The `autocd.ExitWithDirectory()` function in the `autocd-go` library, when integrated with `fzf`, fails silently to replace the current process via `syscall.Exec` when `fzf` is launched from the root directory (`/`). The function returns without an error, and the calling `fzf` process continues execution instead of being replaced by the target shell. This issue does not occur when `fzf` is launched from a user's home directory (`~`).

## Environment
- **OS**: macOS (Darwin)
- **Shell**: zsh (observed)
- **Go Version**: 1.20+ (observed)
- **autocd-go Version**: v0.0.0-20250723123545-8045c46958b6 (or latest `main` branch)
- **Integration**: fzf with `--autocd` flag

## Expected Behavior
When `fzf --autocd` is executed from any directory (including `/`), and a directory is selected, `autocd.ExitWithDirectory()` should:
1.  Successfully replace the `fzf` process with the target shell in the selected directory.
2.  Never return on success.
3.  Only return on failure with a proper error.

## Actual Behavior

### Failure from Root Directory (`/`)
When `fzf` is launched from the root directory, `autocd.ExitWithDirectory()` fails to replace the process. The `fzf` process continues, and the user remains in the original root directory. The `autocd-go` library's internal `cd` command within the temporary script *does* succeed (as indicated by "Directory changed to: ..."), but the process replacement itself does not occur.

```bash
sam@sams-MacBook-Pro / % fzf --autocd
Users/sam/Desktop/Notes/tasks.md
Directory changed to: /Users/sam/Desktop/Notes
sam@sams-MacBook-Pro / %  # ❌ Still in original directory, process not replaced
```

### Success from Home Directory (`~`)
When `fzf` is launched from the home directory, `autocd.ExitWithDirectory()` works as expected, successfully replacing the process and changing the directory.

```bash
sam@sams-MacBook-Pro ~ % fzf --autocd
Desktop/Notes/tasks.md
Directory changed to: /Users/sam/Desktop/Notes
sam@sams-MacBook-Pro Notes %  # ✅ Successfully changed to target directory
```

### Debug Output Issue
Attempts to enable debug output using `AUTOCD_DEBUG=1` (e.g., `AUTOCD_DEBUG=1 fzf --autocd`) do not produce additional debug messages from `autocd-go`, despite `autocd.go` checking for this environment variable. This hinders further investigation.

## Root Cause Analysis (Hypothesis)

The problem is likely an environmental or security-related issue specific to running `syscall.Exec` from the root directory on macOS. Possible factors include:

1.  **macOS Security Policies**: macOS might impose stricter security policies or sandboxing on processes attempting `syscall.Exec` when their current working directory is the root, especially for non-root users. This could lead to a silent failure of `syscall.Exec` that doesn't propagate a standard Go error.
2.  **Process Environment/Inheritance**: Subtle differences in the process environment (e.g., inherited file descriptors, process groups, or capabilities) when launched from `/` might interfere with the `syscall.Exec` operation.
3.  **Shell Initialization from Root**: While the `cd` command in the temporary script works, the subsequent `exec` call to the shell might be affected by how the shell initializes itself when its parent process is launched from `/`.

The lack of debug output from `autocd-go` when `AUTOCD_DEBUG=1` is set is a secondary issue that complicates diagnosis.

## Steps to Reproduce

1.  Ensure `fzf` is built with the `autocd-go` library (e.g., by running `go build` in the `fzf` project root after placing the `autocd-go` folder in the same directory).
2.  Navigate to the root directory: `cd /`
3.  Run `fzf` with the `--autocd` flag: `fzf --autocd`
4.  Select any file or directory.
5.  Observe that the shell prompt remains in `/` and the `fzf` process exits normally, instead of being replaced by the target directory.
6.  Repeat steps 2-5 from your home directory (`cd ~`), and observe that it works correctly.

## Proposed Investigation Steps for `autocd-go` Maintainers

1.  **Address Debug Output**: Investigate why `AUTOCD_DEBUG=1` is not producing output. This might involve:
    *   Verifying that `fmt.Fprintf(os.Stderr, ...)` calls are not being suppressed.
    *   Ensuring the `DebugMode` option is correctly propagated through all relevant functions.
2.  **Detailed `syscall.Exec` Error Handling**: Add more granular error logging around the `syscall.Exec` call in `executeUnixScript`. Specifically, check if `syscall.Exec` returns a non-nil error, and if so, log its value. If it returns `nil` but the process is not replaced, investigate alternative ways to detect this failure (e.g., by checking the process ID after the call, though this is complex with `Exec`).
3.  **Environment Comparison**: Compare the process environment (e.g., `os.Environ()`, `os.Getwd()`, `os.Stdin`/`os.Stdout`/`os.Stderr` file descriptors) just before `syscall.Exec` is called when running from `/` versus `~`.
4.  **Test with Different Shells**: Test the behavior with various shells (bash, zsh, fish) to see if the issue is shell-specific.
5.  **Minimal Reproduction in `autocd-go`**: Create a minimal Go program within the `autocd-go` project that attempts `syscall.Exec` from the root directory to isolate the issue from `fzf`'s context.
