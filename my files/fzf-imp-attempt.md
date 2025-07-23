# FZF AutoCD Integration Implementation Guide

## Overview

This document outlines how to integrate the `autocd-go` library into `fzf` to provide directory inheritance capability via a `--autocd` flag. This would allow fzf to change the shell's working directory to the selected item (if it's a directory) when exiting.

## Library Analysis

### autocd-go Architecture

The `autocd-go` library provides three main entry points:

1. **`ExitWithDirectory(targetPath string) error`** - Basic usage, never returns on success
2. **`ExitWithDirectoryAdvanced(targetPath string, opts *Options) error`** - Advanced usage with configuration options
3. **`ExitWithDirectoryOrFallback(targetPath string, fallback func())`** - Guaranteed exit with fallback handling

#### Key Features:
- Cross-platform support (Windows, macOS, Linux, BSD)
- Multi-shell support (bash, zsh, fish, cmd, PowerShell)
- Security levels (Normal, Strict, Permissive)
- Automatic cleanup of temporary scripts
- Process replacement using `syscall.Exec`
- Structured error handling with recovery information

### FZF Architecture Analysis

From examining the fzf codebase:

- **Entry Point**: `main.go` - calls `fzf.ParseOptions()` then `fzf.Run(options)`
- **Options Structure**: `src/options.go` contains `Options` struct with various flags
- **Flag Parsing**: `ParseOptions()` function handles command-line argument parsing
- **Exit Handling**: Uses standard exit codes defined in `constants.go`:
  - `ExitOk = 0` (successful selection)
  - `ExitNoMatch = 1` (no match found)
  - `ExitError = 2` (error occurred)
  - `ExitBecome = 126` (special exit for become functionality)
  - `ExitInterrupt = 130` (user interrupted)

## Implementation Approach

### 1. Add AutoCD Option to Options Struct

**File: `src/options.go`**

Add to the `Options` struct:

```go
type Options struct {
    // ... existing fields ...
    AutoCD            bool     // Enable autocd functionality
    // ... rest of fields ...
}
```

### 2. Add Command Line Flag Parsing

**File: `src/options.go`** (in `ParseOptions` function)

Add flag parsing for `--autocd`:

```go
case "--autocd":
    opts.AutoCD = true
```

Update the usage string to include the new flag:

```go
const Usage = `...
  DIRECTORY INHERITANCE
    --autocd                 Change shell directory to selected item on exit
...`
```

### 3. Modify Core Exit Logic

**File: `main.go`**

Replace the current exit logic:

```go
func main() {
    // ... existing setup code ...
    
    code, err := fzf.Run(options)
    
    // Handle autocd functionality
    if code == fzf.ExitOk && options.AutoCD {
        handleAutoCD(options)
        // If autocd fails, fall back to normal exit
    }
    
    exit(code, err)
}

func handleAutoCD(options *fzf.Options) {
    // Get the selected item from fzf output
    selectedItem := getSelectedItem(options)
    if selectedItem == "" {
        return
    }
    
    // Check if selected item is a directory
    if !isDirectory(selectedItem) {
        return
    }
    
    // Import autocd library
    // import "github.com/codinganovel/autocd-go"
    
    // Configure autocd options
    autoCDOpts := &autocd.Options{
        SecurityLevel: autocd.SecurityNormal,
        DebugMode:     false, // Could be configurable
    }
    
    // Attempt directory inheritance
    if err := autocd.ExitWithDirectoryAdvanced(selectedItem, autoCDOpts); err != nil {
        // Log error in debug mode but continue with normal exit
        if os.Getenv("FZF_DEBUG") != "" {
            fmt.Fprintf(os.Stderr, "fzf: autocd failed: %v\n", err)
        }
    }
    // If successful, this point is never reached (process replaced)
}

func getSelectedItem(options *fzf.Options) string {
    // This would need to be implemented to capture fzf's selected output
    // May require modifications to the fzf.Run() function to return selection
    // or reading from the output channel/buffer
    return ""
}

func isDirectory(path string) bool {
    info, err := os.Stat(path)
    return err == nil && info.IsDir()
}
```

### 4. Modify fzf.Run() to Support AutoCD

**File: `src/core.go`**

The `Run()` function would need modification to capture the selected item when autocd is enabled:

```go
func Run(opts *Options) (int, string, error) { // Add string return for selected item
    // ... existing implementation ...
    
    // When autocd is enabled, capture the final selection
    var selectedItem string
    if opts.AutoCD {
        // Capture selection logic here
        selectedItem = captureSelection(terminal)
    }
    
    return exitCode, selectedItem, err
}
```

### 5. Dependencies

**File: `go.mod`**

Add the autocd-go dependency:

```go
require (
    // ... existing dependencies ...
    github.com/codinganovel/autocd-go v1.0.0
)
```

## Implementation Challenges and Solutions

### Challenge 1: Capturing Selected Item

**Problem**: fzf's architecture makes it challenging to capture the exact selected item for autocd processing.

**Solution**: 
- Modify the terminal interface to expose the final selection when autocd mode is enabled
- Add a selection capture mechanism in the core Run() function
- Ensure the selection is captured before any exit processing

### Challenge 2: Directory Validation

**Problem**: Need to determine if selected item is a valid directory before attempting autocd.

**Solution**:
- Use `os.Stat()` to check if the selected item is a directory
- Handle cases where the selected item might be a relative path
- Integrate with autocd-go's built-in path validation

### Challenge 3: Error Handling

**Problem**: autocd failures should not break fzf's normal operation.

**Solution**:
- Implement graceful fallback to normal exit on autocd failure
- Use structured error handling from autocd-go library
- Provide debug output for troubleshooting when enabled

### Challenge 4: Multi-selection Support

**Problem**: Handling autocd when multiple items are selected.

**Solution**:
- When `--multi` is used with `--autocd`, use the first selected directory
- Skip autocd if no directories are in the selection
- Consider adding a flag to control this behavior

## Advanced Features

### 1. Security Level Configuration

Add additional flags for security configuration:

```bash
--autocd-security=LEVEL    # normal, strict, permissive
```

### 2. Debug Mode

Enable autocd debug output:

```bash
--autocd-debug            # Enable autocd debug logging
```

### 3. Shell Override

Allow shell override for autocd:

```bash
--autocd-shell=SHELL      # Override shell detection
```

## Testing Strategy

### Unit Tests

1. **Option Parsing**: Test `--autocd` flag parsing
2. **Directory Detection**: Test directory validation logic
3. **Error Handling**: Test graceful fallback on autocd failures

### Integration Tests

1. **Basic Usage**: Test `fzf --autocd` with directory selection
2. **Multi-select**: Test behavior with multiple selections
3. **Non-directories**: Test selection of files (should skip autocd)
4. **Error Cases**: Test invalid directories, permission issues

### Platform Testing

1. **Cross-platform**: Test on Linux, macOS, Windows
2. **Multi-shell**: Test with bash, zsh, fish, PowerShell
3. **Edge Cases**: Test with special characters in paths

## Usage Examples

### Basic Directory Navigation

```bash
# Navigate directories and inherit final location
find . -type d | fzf --autocd

# Use with fd for modern directory navigation
fd -t d | fzf --autocd

# Browse and navigate to any directory
ls -la | fzf --autocd
```

### Advanced Usage

```bash
# With security and debug options
fzf --autocd --autocd-security=strict --autocd-debug

# Combined with other fzf features
fzf --multi --autocd --preview 'ls -la {}'
```

## Implementation Timeline

1. **Phase 1**: Basic autocd flag and option parsing
2. **Phase 2**: Core autocd integration and selection capture
3. **Phase 3**: Error handling and fallback mechanisms
4. **Phase 4**: Advanced features and security options
5. **Phase 5**: Testing and documentation

## Conclusion

Integrating autocd-go into fzf would provide powerful directory inheritance capability, solving the long-standing Unix limitation of losing your location when exiting navigation tools. The implementation leverages fzf's existing architecture while adding minimal complexity, ensuring backward compatibility and graceful error handling.

The key to success is careful integration with fzf's selection mechanism and robust error handling to ensure that autocd failures don't disrupt normal fzf operation.