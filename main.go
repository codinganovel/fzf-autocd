package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/codinganovel/autocd-go"
	fzf "github.com/junegunn/fzf/src"
	"github.com/junegunn/fzf/src/protector"
	"github.com/junegunn/fzf/src/tui"
	"github.com/junegunn/fzf/src/util"
)

var version = "0.64"
var revision = "devel"

//go:embed shell/key-bindings.bash
var bashKeyBindings []byte

//go:embed shell/completion.bash
var bashCompletion []byte

//go:embed shell/key-bindings.zsh
var zshKeyBindings []byte

//go:embed shell/completion.zsh
var zshCompletion []byte

//go:embed shell/key-bindings.fish
var fishKeyBindings []byte

//go:embed man/man1/fzf.1
var manPage []byte

func printScript(label string, content []byte) {
	fmt.Println("### " + label + " ###")
	fmt.Println(strings.TrimSpace(string(content)))
	fmt.Println("### end: " + label + " ###")
}

func exit(code int, err error) {
	if code == fzf.ExitError && err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	os.Exit(code)
}

func main() {
	protector.Protect()

	options, err := fzf.ParseOptions(true, os.Args[1:])
	if err != nil {
		exit(fzf.ExitError, err)
		return
	}
	if options.Bash {
		printScript("key-bindings.bash", bashKeyBindings)
		printScript("completion.bash", bashCompletion)
		return
	}
	if options.Zsh {
		printScript("key-bindings.zsh", zshKeyBindings)
		printScript("completion.zsh", zshCompletion)
		return
	}
	if options.Fish {
		printScript("key-bindings.fish", fishKeyBindings)
		fmt.Println("fzf_key_bindings")
		return
	}
	if options.Help {
		fmt.Print(fzf.Usage)
		return
	}
	if options.Version {
		if len(revision) > 0 {
			fmt.Printf("%s (%s)\n", version, revision)
		} else {
			fmt.Println(version)
		}
		return
	}
	if options.Man {
		file := fzf.WriteTemporaryFile([]string{string(manPage)}, "\n")
		if len(file) == 0 {
			fmt.Print(string(manPage))
			return
		}
		defer os.Remove(file)
		cmd := exec.Command("man", file)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			fmt.Print(string(manPage))
		}
		return
	}

	code, selectedItem, err := fzf.Run(options)
	
	// Handle autocd functionality
	if code == fzf.ExitOk && options.AutoCD && selectedItem != "" {
		handleAutoCD(selectedItem)
		// If autocd fails, fall back to normal exit
	}
	
	exit(code, err)
}

func handleAutoCD(selectedItem string) {
	var targetDir string
	if isDirectory(selectedItem) {
		targetDir = selectedItem
	} else {
		targetDir = filepath.Dir(selectedItem)
	}

	// Fix: Set up stdin properly before calling autocd, just like fzf's become action
	// This ensures the terminal file descriptor is correctly configured regardless of
	// the working directory from which fzf was launched
	if ttyin, err := tui.TtyIn(tui.DefaultTtyDevice); err == nil {
		util.SetStdin(ttyin)
	}

	autocd.ExitWithDirectoryOrFallback(targetDir, func() {
		fmt.Fprintf(os.Stderr, "fzf: autocd failed, falling back to normal exit\n")
		os.Exit(0)
	})
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
