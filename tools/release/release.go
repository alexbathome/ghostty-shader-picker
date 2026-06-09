// A small tool to help with releasing a new version of ghostty-shader-picker.
// It runs the relevant git submodule pulls, generate, tagging, and pushing.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Check we're on main.
	branch, err := gitOutput("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("checking branch: %w", err)
	}
	if branch != "main" {
		return fmt.Errorf("must be on main branch (currently on %q)", branch)
	}

	// 2. Check for uncommitted changes.
	diff, err := gitOutput("status", "--porcelain")
	if err != nil {
		return fmt.Errorf("checking git status: %w", err)
	}
	if diff != "" {
		return fmt.Errorf("working tree has uncommitted changes — commit or stash them first:\n%s", diff)
	}

	// 3. Show the latest tag.
	latestTag, err := gitOutput("describe", "--tags", "--abbrev=0")
	if err != nil {
		latestTag = "(none)"
	}
	fmt.Printf("Latest tag: %s\n", latestTag)

	// 4. Prompt for the next tag.
	fmt.Print("Next tag: ")
	nextTag := strings.TrimSpace(readLine())
	if nextTag == "" {
		return fmt.Errorf("tag cannot be empty")
	}

	// 5. Confirm.
	fmt.Printf("This will tag %q and push to origin. Continue? [y/N] ", nextTag)
	answer := strings.TrimSpace(strings.ToLower(readLine()))
	if answer != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	// 6. Sync the submodule.
	fmt.Println("Updating submodule...")
	if err := gitRun("submodule", "update", "--remote", "--merge"); err != nil {
		return fmt.Errorf("updating submodule: %w", err)
	}

	// 7. Vendor shaders via go generate.
	fmt.Println("Running go generate...")
	cmd := exec.Command("go", "generate", "./internal/picker/...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go generate: %w", err)
	}

	// 8. Stage and commit if there are changes.
	postGenDiff, _ := gitOutput("status", "--porcelain")
	if postGenDiff != "" {
		fmt.Println("Committing vendored shader updates...")
		if err := gitRun("add", "internal/picker/ghostty-shaders-dist/", "internal/picker/ghostty-shaders"); err != nil {
			return fmt.Errorf("staging changes: %w", err)
		}
		if err := gitRun("commit", "-m", "chore: sync vendored shaders for "+nextTag); err != nil {
			return fmt.Errorf("committing changes: %w", err)
		}
	}

	// 9. Create the tag.
	fmt.Printf("Tagging %s...\n", nextTag)
	if err := gitRun("tag", nextTag); err != nil {
		return fmt.Errorf("creating tag: %w", err)
	}

	// 10. Push branch and tag.
	fmt.Println("Pushing to origin...")
	if err := gitRun("push", "origin", "main"); err != nil {
		return fmt.Errorf("pushing branch: %w", err)
	}
	if err := gitRun("push", "origin", nextTag); err != nil {
		return fmt.Errorf("pushing tag: %w", err)
	}

	fmt.Printf("Released %s\n", nextTag)
	return nil
}

func gitOutput(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	return strings.TrimSpace(string(out)), err
}

func gitRun(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func readLine() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text()
}
