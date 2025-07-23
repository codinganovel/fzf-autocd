# Bug Report: autocd-go ExitWithDirectory() Process Replacement Failure

## Summary
The primary `ExitWithDirectory()` function in autocd-go fails silently during process replacement on macOS, causing the function to return instead of replacing the current process as documented. The fallback function `ExitWithDirectoryOrFallback()` works correctly, masking the underlying issue.

## Environment
- **OS**: macOS (Darwin 25.0.0)
- **Shell**: zsh
- **Go Version**: 1.20+
- **autocd-go Version**: v0.0.0-20250723123545-8045c46958b6 (latest from main branch)
- **Integration**: fzf with `--autocd` flag

## Expected Behavior
According to the library documentation:

### `ExitWithDirectory()` should:
1. **Never return on success** (process is replaced via `syscall.Exec`)
2. **Only return on failure** with a proper error
3. Be the **primary interface** for directory inheritance

### Debug output should:
1. Only appear when `AUTOCD_DEBUG=1` environment variable is set
2. Not appear in normal operation

## Actual Behavior

### Primary Issue: `ExitWithDirectory()` Fails Silently
```bash
sam@sams-MacBook-Pro / % fzf --autocd
Users/sam/Desktop/Notes/Archive/tasks.md
DEBUG: autocd selected item: 'Users/sam/Desktop/Notes/Archive/tasks.md'
DEBUG: 'Users/sam/Desktop/Notes/Archive/tasks.md' is a file, using parent directory: 'Users/sam/Desktop/Notes/Archive'
DEBUG: attempting autocd to: 'Users/sam/Desktop/Notes/Archive'
Directory changed to: /Users/sam/Desktop/Notes/Archive
sam@sams-MacBook-Pro / %  # ❌ Still in original directory, process not replaced
```

**Problems observed:**
1. ❌ Function returns instead of replacing process
2. ❌ No error returned despite failure
3. ❌ User remains in original directory (`/`) instead of target (`/Users/sam/Desktop/Notes/Archive`)
4. ❌ Debug output appears without `AUTOCD_DEBUG` environment variable

### Workaround: `ExitWithDirectoryOrFallback()` Works
```bash
sam@sams-MacBook-Pro / % fzf --autocd  # Using ExitWithDirectoryOrFallback()
Users/sam/Desktop/Notes/Archive/tasks.md
DEBUG: autocd selected item: 'Users/sam/Desktop/Notes/Archive/tasks.md'
DEBUG: 'Users/sam/Desktop/Notes/Archive/tasks.md' is a file, using parent directory: 'Users/sam/Desktop/Notes/Archive'
DEBUG: attempting autocd to: 'Users/sam/Desktop/Notes/Archive'
Directory changed to: /Users/sam/Desktop/Notes/Archive
sam@sams-MacBook-Pro Archive %  # ✅ Successfully changed to target directory
```

## Code Implementation

### Failing Implementation (using ExitWithDirectory):
```go
func handleAutoCD(selectedItem string) {
    var targetDir string
    if isDirectory(selectedItem) {
        targetDir = selectedItem
    } else {
        targetDir = filepath.Dir(selectedItem)
    }
    
    // This should never return on success, but it does
    if err := autocd.ExitWithDirectory(targetDir); err != nil {
        fmt.Fprintf(os.Stderr, "fzf: autocd failed: %v\n", err)
    }
    // ❌ This point should never be reached on success, but it is
}
```

### Working Implementation (using ExitWithDirectoryOrFallback):
```go
func handleAutoCD(selectedItem string) {
    var targetDir string
    if isDirectory(selectedItem) {
        targetDir = selectedItem
    } else {
        targetDir = filepath.Dir(selectedItem)
    }
    
    // This works correctly - never returns
    autocd.ExitWithDirectoryOrFallback(targetDir, func() {
        fmt.Fprintf(os.Stderr, "fzf: autocd fallback executed\n")
        os.Exit(0)
    })
    // ✅ This point is never reached
}
```

## Root Cause Analysis

### Issue 1: Silent Process Replacement Failure
The `syscall.Exec` call in `ExitWithDirectory()` appears to be failing but not returning an error. This suggests:

1. **Script generation succeeds** (we see "Directory changed to: ..." message)
2. **Process replacement fails** (`syscall.Exec` doesn't work)
3. **Error handling is inadequate** (no error returned despite failure)

### Issue 2: Debug Output Always Enabled
The library is printing debug messages regardless of the `AUTOCD_DEBUG` environment variable state. This suggests either:

1. Debug mode is being force-enabled in our integration
2. The library's debug mode detection is not working properly
3. Some messages are not properly gated behind debug mode checks

## Investigation Needed

### Process Replacement Investigation
1. **Check `syscall.Exec` return values** in `executeUnixScript()`
2. **Verify script execution permissions** (should be 0700)
3. **Test script execution manually** to isolate the issue
4. **Check if shell detection is working correctly**

### Debug Output Investigation
1. **Verify `AUTOCD_DEBUG` environment variable handling**
2. **Check if debug messages are properly conditional**
3. **Identify which component is printing unconditional debug output**

## Minimal Reproduction Case

```go
package main

import (
    "fmt"
    "os"
    "github.com/codinganovel/autocd-go"
)

func main() {
    fmt.Printf("Current directory: %s\n", mustGetWd())
    
    // This should replace the process and never return
    if err := autocd.ExitWithDirectory("/tmp"); err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    
    // This should never execute
    fmt.Printf("❌ BUG: Function returned instead of replacing process\n")
    fmt.Printf("Still in directory: %s\n", mustGetWd())
}

func mustGetWd() string {
    wd, _ := os.Getwd()
    return wd
}
```

**Expected:** Process is replaced, user gets shell in `/tmp`  
**Actual:** Function returns, prints bug message, user stays in original directory

## Impact
- **High**: Primary library function doesn't work as documented
- **Workaround exists**: `ExitWithDirectoryOrFallback()` provides working functionality
- **User confusion**: Debug output appears unexpectedly
- **Integration complexity**: Developers must use secondary function instead of primary interface

## Proposed Investigation Steps
1. Add detailed logging to `executeUnixScript()` function
2. Check `syscall.Exec` error returns and failure modes
3. Test script execution manually outside of Go
4. Verify shell detection and script generation
5. Fix debug output conditional logic
6. Add proper error reporting for process replacement failures

## Integration Context
This issue was discovered while implementing `--autocd` functionality in fzf. The integration works correctly with `ExitWithDirectoryOrFallback()` but the primary `ExitWithDirectory()` function fails silently, which goes against the library's documented behavior and design intent.