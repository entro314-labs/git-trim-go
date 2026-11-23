// Package ui provides user interface utilities for git-trim.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	ColorBranch    = color.New(color.FgCyan, color.Bold)
	ColorRemote    = color.New(color.FgMagenta)
	ColorWarning   = color.New(color.FgYellow)
	ColorError     = color.New(color.FgRed, color.Bold)
	ColorSuccess   = color.New(color.FgGreen)
	ColorInfo      = color.New(color.FgBlue)
	ColorPreserved = color.New(color.FgWhite, color.Faint)
	ColorDryRun    = color.New(color.FgYellow, color.Bold)
)

// Confirm prompts the user for confirmation
func Confirm(message string) (bool, error) {
	fmt.Printf("%s [y/N]: ", message)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// ConfirmWithDefault prompts the user for confirmation with a default value
func ConfirmWithDefault(message string, defaultValue bool) (bool, error) {
	defaultStr := "[y/N]"
	if defaultValue {
		defaultStr = "[Y/n]"
	}

	fmt.Printf("%s %s: ", message, defaultStr)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "" {
		return defaultValue, nil
	}

	return response == "y" || response == "yes", nil
}

// PrintHeader prints a section header
func PrintHeader(title string) {
	fmt.Println()
	_, _ = ColorInfo.Println("═══════════════════════════════════════════════════")
	_, _ = ColorInfo.Printf("  %s\n", title)
	_, _ = ColorInfo.Println("═══════════════════════════════════════════════════")
	fmt.Println()
}

// PrintSubHeader prints a sub-section header
func PrintSubHeader(title string) {
	fmt.Println()
	_, _ = ColorInfo.Printf("─── %s ───\n", title)
}

// PrintBranch prints a branch name with formatting
func PrintBranch(branchName string) string {
	return ColorBranch.Sprint(branchName)
}

// PrintRemote prints a remote name with formatting
func PrintRemote(remoteName string) string {
	return ColorRemote.Sprint(remoteName)
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	_, _ = ColorWarning.Printf("⚠ %s\n", message)
}

// PrintError prints an error message
func PrintError(message string) {
	_, _ = ColorError.Printf("✗ %s\n", message)
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	_, _ = ColorSuccess.Printf("✓ %s\n", message)
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	_, _ = ColorInfo.Printf("ℹ %s\n", message)
}

// PrintDryRun prints a dry-run notice
func PrintDryRun() {
	fmt.Println()
	_, _ = ColorDryRun.Println("╔═══════════════════════════════════════════════════╗")
	_, _ = ColorDryRun.Println("║              DRY RUN - No changes made            ║")
	_, _ = ColorDryRun.Println("╚═══════════════════════════════════════════════════╝")
	fmt.Println()
}

// PrintNoChanges prints a message when there's nothing to do
func PrintNoChanges() {
	PrintInfo("No branches to delete. Everything is clean!")
}

// FormatBranchList formats a list of branch names
func FormatBranchList(branches []string) string {
	if len(branches) == 0 {
		return "(none)"
	}
	return strings.Join(branches, ", ")
}
