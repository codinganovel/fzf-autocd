# FZF Autocd Bug: Debug Report

## Problem Statement
The `autocd` functionality in `fzf` is intended to replace the current process and change the directory upon selection. While it works correctly when `fzf` is launched from the home directory (`~`), it consistently fails when launched from the root directory (`/`). The `fzf` process simply continues execution without changing the directory.

## Key Observations and Debugging Attempts

1.  **Standalone `autocd-go` Test Application:**
    *   A minimal Go application directly utilizing the `autocd-go` library's `ExitWithDirectory()` function was created and tested.
    *   This standalone application successfully performs process replacement and directory change when launched from *both* the home directory (`~`) and the root directory (`/`).
    *   **Conclusion:** This confirms that the `autocd-go` library itself and the underlying `syscall.Exec` mechanism are not inherently flawed or the cause of the issue.

2.  **`fzf` Behavior:**
    *   When `fzf` is launched from `~`, the `autocd` functionality works as expected.
    *   When `fzf` is launched from `/`, the `autocd` functionality silently fails; `fzf` remains active and does not change the directory.

3.  **Debugging Challenges:**
    *   Direct `fmt.Fprintf(os.Stderr, ...)` debug logging from within `fzf` (specifically in `src/terminal.go`) does not appear in the terminal. This suggests that `fzf` might be redirecting or suppressing `stderr` output, making traditional debugging difficult.
    *   Attempts to implement file-based logging were made but were complicated by the precision required for `replace` operations on complex string literals within the Go source code.

## Current Hypothesis
Given that the `autocd-go` library works correctly in isolation from both `~` and `/`, the problem is strongly suspected to lie within `fzf`'s internal state management or terminal handling when it is initialized or run from the root directory (`/`). There might be subtle differences in file descriptor handling, terminal raw mode settings, or process group management that prevent `syscall.Exec` from succeeding only in the `/` context within `fzf`.

## Next Steps (Proposed)
Further investigation is required to understand `fzf`'s internal state and terminal interactions when launched from `/`. This may involve:
*   More robust file-based logging to capture detailed information about file descriptors, terminal settings, and process environment variables at critical points in `fzf`'s execution path.
*   Careful analysis of `fzf`'s terminal initialization and restoration logic, particularly in `src/tui/` and `src/terminal.go`, to identify any code paths that behave differently based on the current working directory.
*   Potentially using `strace` or `dtrace` (or equivalent system tracing tools) if direct code-level debugging continues to be challenging, to observe system calls made by `fzf` when launched from `~` versus `/`.
