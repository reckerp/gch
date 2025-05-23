package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// IsGitRepo checks if the current directory is a git repository
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}

// promptForStash shows an interactive prompt asking the user if they want to stash changes
func promptForStash() (bool, error) {
	model := createStashPromptModel()
	p := tea.NewProgram(model)
	result, err := p.Run()
	if err != nil {
		return false, err
	}

	// The result is the model, and we can check which option was selected
	if m, ok := result.(*stashPromptModel); ok {
		return m.cursor == 0, nil // true if "Stash changes" was selected
	}
	return false, errors.New("unexpected result type from stash prompt")
}

// stashChanges stashes the current changes
func stashChanges() error {
	return execGitCommand("stash", "push", "-m", "Auto-stashed by gch")
}

// execGitCommandWithOutput executes a git command and returns its output
func execGitCommandWithOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// SmartCheckout implements smart branch checkout functionality
func SmartCheckout(pattern string, createBranch bool, force bool, stash bool, debug bool) error {
	if pattern == "" {
		// If no pattern provided, switch to the previous branch
		return execGitCommand("checkout", "-")
	}

	// If createBranch is true, create and checkout a new branch
	if createBranch {
		fmt.Printf("Creating and checking out new branch: %s\n", pattern)
		args := []string{"checkout", "-b", pattern}
		if force {
			args = append(args, "-f")
		}
		return execGitCommand(args...)
	}

	// Get all branches (local and remote)
	branches, err := getAllBranches()
	if err != nil {
		return err
	}

	if debug {
		fmt.Printf("Found %d branches\n", len(branches))
	}

	// If no branches exist and no pattern provided, suggest creating a new branch
	if len(branches) == 0 {
		return fmt.Errorf("no branches found. Use -b flag to create a new branch")
	}

	// Convert to branchMatch objects and score them
	var matches []branchMatch
	for _, branch := range branches {
		score := calcMatchScore(branch.Name, pattern)
		if score > 0 { // Only add if there's some match
			matches = append(matches, branchMatch{
				name:    branch.Name,
				isLocal: branch.IsLocal,
				score:   score,
			})
		}
	}

	if len(matches) == 0 {
		// If no matches found, try fetching and searching again
		if err := execGitCommand("fetch", "--quiet"); err != nil {
			return fmt.Errorf("failed to fetch remote branches: %w", err)
		}

		branches, err = getAllBranches()
		if err != nil {
			return err
		}

		matches = nil
		for _, branch := range branches {
			score := calcMatchScore(branch.Name, pattern)
			if score > 0 {
				matches = append(matches, branchMatch{
					name:    branch.Name,
					isLocal: branch.IsLocal,
					score:   score,
				})
			}
		}

		if len(matches) == 0 {
			return errors.New("no branches match '" + pattern + "'")
		}
	}

	// Sort matches by score (higher is better)
	sortMatches(matches)

	if debug {
		fmt.Printf("Found %d matches:\n", len(matches))
		for i, match := range matches {
			fmt.Printf("%d. %s (score: %d, local: %v)\n", i+1, match.name, match.score, match.isLocal)
		}
	}

	bestMatch := matches[0]
	// If we have a single match or one match is significantly better than others
	if len(matches) == 1 || (len(matches) > 1 && bestMatch.score > matches[1].score*2) {
		// Single match or one match is significantly better than others
		if bestMatch.isLocal {
			// Local branch
			fmt.Printf("Checking out local branch: %s\n", bestMatch.name)

			// If stash flag is set, always stash changes
			if stash {
				if err := stashChanges(); err != nil {
					return fmt.Errorf("failed to stash changes: %w", err)
				}
			} else if !force {
				// Only check for conflicts if not forcing and not stashing
				output, err := execGitCommandWithOutput("checkout", bestMatch.name)
				if err != nil {
					if strings.Contains(output, "error: Your local changes to the following files would be overwritten by checkout") ||
						strings.Contains(output, "error: The following untracked working tree files would be overwritten by checkout") {
						// Checkout would fail, ask about stashing
						stash, err := promptForStash()
						if err != nil {
							return err
						}
						if stash {
							if err := stashChanges(); err != nil {
								return fmt.Errorf("failed to stash changes: %w", err)
							}
							// Try the checkout again after stashing
							return execGitCommand("checkout", bestMatch.name)
						} else {
							return errors.New("checkout aborted")
						}
					}
					return fmt.Errorf("git checkout failed: %s", output)
				}
				return nil
			}

			args := []string{"checkout", bestMatch.name}
			if force {
				args = append(args, "-f")
			}
			return execGitCommand(args...)
		} else {
			// Remote branch
			fmt.Printf("Creating local branch from remote: %s\n", bestMatch.name)

			// If stash flag is set, always stash changes
			if stash {
				if err := stashChanges(); err != nil {
					return fmt.Errorf("failed to stash changes: %w", err)
				}
			} else if !force {
				// Only check for conflicts if not forcing and not stashing
				output, err := execGitCommandWithOutput("checkout", "-b", bestMatch.name, "origin/"+bestMatch.name)
				if err != nil {
					if strings.Contains(output, "error: Your local changes to the following files would be overwritten by checkout") ||
						strings.Contains(output, "error: The following untracked working tree files would be overwritten by checkout") {
						// Checkout would fail, ask about stashing
						stash, err := promptForStash()
						if err != nil {
							return err
						}
						if stash {
							if err := stashChanges(); err != nil {
								return fmt.Errorf("failed to stash changes: %w", err)
							}
							// Try the checkout again after stashing
							return execGitCommand("checkout", "-b", bestMatch.name, "origin/"+bestMatch.name)
						} else {
							return errors.New("checkout aborted")
						}
					}
					return fmt.Errorf("git checkout failed: %s", output)
				}
				return nil
			}

			args := []string{"checkout", "-b", bestMatch.name, "origin/" + bestMatch.name}
			if force {
				args = append(args, "-f")
			}
			return execGitCommand(args...)
		}
	} else {
		// Multiple matches with similar scores - start interactive selector
		fmt.Printf("Multiple matches found. Starting interactive selector...\n\n")

		// Create a filtered model with only the matching branches
		model := createFilteredBranchModel(matches, debug)
		p := tea.NewProgram(model)
		_, err = p.Run()
		return err
	}
}

// getBranches returns a list of branches (local or remote)
func getBranches(remote bool) ([]string, error) {
	var args []string
	if remote {
		args = []string{"branch", "-r", "--format=%(refname:short)"}
	} else {
		args = []string{"branch", "--format=%(refname:short)"}
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		// If the error is due to no branches, return empty slice instead of error
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string

	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if branch == "" {
			continue
		}

		if remote {
			// Skip HEAD reference
			if strings.Contains(branch, "HEAD") {
				continue
			}
			// Remove the 'origin/' prefix
			branch = strings.TrimPrefix(branch, "origin/")
		}

		result = append(result, branch)
	}

	return result, nil
}

// execGitCommand executes a git command with the given arguments
func execGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
