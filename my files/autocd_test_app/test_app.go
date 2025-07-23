package main

import (
"fmt"
"os"
"github.com/codinganovel/autocd-go"
)

func main() {
if len(os.Args) != 2 {
fmt.Fprintf(os.Stderr, "Usage: %s <directory>\n", os.Args[0])
fmt.Fprintf(os.Stderr, "Example: %s /tmp\n", os.Args[0])
os.Exit(1)
}

targetDir := os.Args[1]

fmt.Printf("Current directory: %s\n", getCurrentDir())
fmt.Printf("Target directory: %s\n", targetDir)
fmt.Printf("About to call ExitWithDirectory()...\n")

// This should replace the process and never return
if err := autocd.ExitWithDirectory(targetDir); err != nil {
fmt.Printf("Error: %v\n", err)
os.Exit(1)
}

// This should NEVER execute if process replacement works
fmt.Printf("❌ BUG: Process was NOT replaced - this line should never be reached!\n")
fmt.Printf("Still in directory: %s\n", getCurrentDir())
}

func getCurrentDir() string {
wd, err := os.Getwd()
if err != nil {
return "unknown"
}
return wd
}
