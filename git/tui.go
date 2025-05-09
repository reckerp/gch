package git

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

// Model represents the TUI model for branch selection
type branchModel struct {
	branches        []Branch
	filteredIdx     []int
	selected        int
	query           string
	width           int
	height          int
	showRemotes     bool
	debugMode       bool
	showStashPrompt bool
	stashPrompt     *stashPromptModel
}

// Branch represents a git branch
type Branch struct {
	Name    string
	IsLocal bool
	Current bool
}

// String returns the string representation of a branch
func (b Branch) String() string {
	if b.Current {
		return "* " + b.Name
	}

	if !b.IsLocal {
		return "  " + b.Name + " (remote)"
	}

	return "  " + b.Name
}

// Initial model
func initialBranchModel(debugMode bool) (branchModel, error) {
	// Fetch latest remote information
	if err := execGitCommand("fetch", "--quiet"); err != nil {
		return branchModel{}, fmt.Errorf("failed to fetch remote branches: %w", err)
	}

	// Get branches
	branches, err := getAllBranches()
	if err != nil {
		return branchModel{}, err
	}

	model := branchModel{
		branches:    branches,
		selected:    0,
		query:       "",
		width:       80,
		height:      20,
		showRemotes: true,
		debugMode:   debugMode,
	}

	// Initial filter (show all branches)
	model.filter("")

	return model, nil
}

// Filter branches based on query
func (m *branchModel) filter(query string) {
	m.query = query

	// If no query, show all branches
	if query == "" {
		m.filteredIdx = make([]int, len(m.branches))
		for i := range m.branches {
			m.filteredIdx[i] = i
		}
		return
	}

	// Create slice of branch names for fuzzy matching
	var names []string
	for _, b := range m.branches {
		names = append(names, b.Name)
	}

	// Perform fuzzy matching
	matches := fuzzy.Find(query, names)
	m.filteredIdx = make([]int, len(matches))
	for i, match := range matches {
		m.filteredIdx[i] = match.Index
	}

	// Reset selected item if out of range
	if len(m.filteredIdx) > 0 && m.selected >= len(m.filteredIdx) {
		m.selected = 0
	}
}

// Init initializes the model
func (m branchModel) Init() tea.Cmd {
	return nil
}

// Update handles user input
func (m branchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If showing stash prompt, handle it first
	if m.showStashPrompt {
		if m.stashPrompt == nil {
			m.stashPrompt = createStashPromptModel()
		}

		model, cmd := m.stashPrompt.Update(msg)
		if model, ok := model.(*stashPromptModel); ok {
			m.stashPrompt = model
			if model.selected {
				// User made a choice
				if model.cursor == 0 {
					// User chose to stash
					return m, tea.Sequence(
						tea.ExecProcess(exec.Command("git", "stash", "push", "-m", "Auto-stashed by gch"), func(err error) tea.Msg {
							if err != nil {
								return err
							}
							// After stashing, try the checkout again
							selectedBranch := m.branches[m.filteredIdx[m.selected]]
							var args []string
							if selectedBranch.IsLocal {
								args = []string{"checkout", selectedBranch.Name}
							} else {
								args = []string{"checkout", "-b", selectedBranch.Name, "origin/" + selectedBranch.Name}
							}
							return exec.Command("git", args...)
						}),
						tea.Quit,
					)
				} else {
					// User chose to abort
					return m, tea.Quit
				}
			}
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "enter":
			if len(m.filteredIdx) > 0 {
				selectedBranch := m.branches[m.filteredIdx[m.selected]]
				var args []string
				if selectedBranch.IsLocal {
					args = []string{"checkout", selectedBranch.Name}
				} else {
					args = []string{"checkout", "-b", selectedBranch.Name, "origin/" + selectedBranch.Name}
				}

				// Try the checkout to see if it would fail
				cmd := exec.Command("git", args...)
				output, err := cmd.CombinedOutput()
				if err != nil {
					errorMsg := string(output)
					if strings.Contains(errorMsg, "error: Your local changes to the following files would be overwritten by checkout") ||
						strings.Contains(errorMsg, "error: The following untracked working tree files would be overwritten by checkout") {
						// Checkout would fail, show stash prompt
						m.showStashPrompt = true
						return m, nil
					}
					// If it's a different error, return it
					return m, tea.Sequence(
						tea.ExecProcess(exec.Command("git", args...), func(err error) tea.Msg {
							return err
						}),
						tea.Quit,
					)
				}

				// If checkout succeeded, we're done
				return m, tea.Quit
			}

		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}

		case "down", "j":
			if m.selected < len(m.filteredIdx)-1 {
				m.selected++
			}

		case "backspace":
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.filter(m.query)
			}

		default:
			if len(msg.String()) == 1 {
				m.query += msg.String()
				m.filter(m.query)
			}
		}
	}

	return m, nil
}

// View renders the UI
func (m branchModel) View() string {
	if m.showStashPrompt {
		if m.stashPrompt == nil {
			m.stashPrompt = createStashPromptModel()
		}
		return m.stashPrompt.View()
	}

	var sb strings.Builder

	// Show search query
	sb.WriteString(fmt.Sprintf("Search: %s\n\n", m.query))

	// Show branches
	visibleCount := 0
	for i, idx := range m.filteredIdx {
		if visibleCount >= m.height-5 {
			sb.WriteString("  (more branches not shown)\n")
			break
		}

		branch := m.branches[idx]

		if i == m.selected {
			// Highlight selected branch
			sb.WriteString("> " + highlightMatches(branch.String(), m.query) + "\n")
		} else {
			sb.WriteString("  " + highlightMatches(branch.String(), m.query) + "\n")
		}

		visibleCount++
	}

	if len(m.filteredIdx) == 0 {
		sb.WriteString("\nNo matching branches found\n")
	}

	// Help text
	sb.WriteString("\nArrow keys/j/k to navigate, Enter to select, q to quit\n")

	return sb.String()
}

// highlightMatches highlights matching characters in a string
func highlightMatches(s, query string) string {
	if query == "" {
		return s
	}

	var result string

	// Simple case-insensitive highlighting
	lowerStr := strings.ToLower(s)
	lowerQuery := strings.ToLower(query)

	lastIdx := 0
	for i := range len(lowerStr) {
		if strings.HasPrefix(lowerStr[i:], lowerQuery) {
			// Add text before match
			result += s[lastIdx:i]
			// Add highlighted match
			result += s[i : i+len(query)]
			lastIdx = i + len(query)
		}
	}

	// Add remaining text
	result += s[lastIdx:]

	return result
}

// execGitForTUI executes a git checkout command in a way that works with bubbletea
func execGitForTUI(branch Branch) tea.Cmd {
	var args []string
	if branch.IsLocal {
		args = []string{"checkout", branch.Name}
	} else {
		args = []string{"checkout", "-b", branch.Name, "origin/" + branch.Name}
	}

	// First try the checkout to see if it would fail
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errorMsg := string(output)
		if strings.Contains(errorMsg, "error: Your local changes to the following files would be overwritten by checkout") ||
			strings.Contains(errorMsg, "error: The following untracked working tree files would be overwritten by checkout") {
			// Checkout would fail, ask about stashing
			return tea.Sequence(
				tea.ExecProcess(exec.Command("git", "stash", "push", "-m", "Auto-stashed by gch"), func(err error) tea.Msg {
					if err != nil {
						return err
					}
					// After stashing, try the checkout again
					return exec.Command("git", args...)
				}),
				tea.Quit,
			)
		}
		// If it's a different error, return it
		return tea.Sequence(
			tea.ExecProcess(exec.Command("git", args...), func(err error) tea.Msg {
				return err
			}),
			tea.Quit,
		)
	}

	// If checkout succeeded, we're done
	return tea.Quit
}

// ShowInteractiveBranchSelector shows an interactive branch selector
func ShowInteractiveBranchSelector(debugMode bool) error {
	// Check if we're in an empty repository
	cmd := exec.Command("git", "rev-parse", "HEAD")
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
			return fmt.Errorf("empty repository. Use -b flag to create a new branch")
		}
		return err
	}

	model, err := initialBranchModel(debugMode)
	if err != nil {
		return err
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// getAllBranches returns all branches, both local and remote
func getAllBranches() ([]Branch, error) {
	// Get current branch
	currentBranch, err := getCurrentBranch()
	if err != nil {
		return nil, err
	}

	// Get local branches
	localBranches, err := getBranches(false)
	if err != nil {
		return nil, err
	}

	// Get remote branches
	remoteBranches, err := getBranches(true)
	if err != nil {
		return nil, err
	}

	// Create a map to avoid duplicates
	branchMap := make(map[string]Branch)

	// Add local branches
	for _, name := range localBranches {
		branchMap[name] = Branch{
			Name:    name,
			IsLocal: true,
			Current: name == currentBranch,
		}
	}

	// Add remote branches that don't have a local counterpart
	for _, name := range remoteBranches {
		if _, exists := branchMap[name]; !exists {
			branchMap[name] = Branch{
				Name:    name,
				IsLocal: false,
				Current: false,
			}
		}
	}

	// Convert map to slice
	var result []Branch
	for _, branch := range branchMap {
		result = append(result, branch)
	}

	return result, nil
}

// getCurrentBranch returns the current branch name
func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		// In an empty repository, return empty string instead of error
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// stashPromptModel represents the model for the stash prompt
type stashPromptModel struct {
	choices  []string
	cursor   int
	selected bool
}

// createStashPromptModel creates a new stash prompt model
func createStashPromptModel() *stashPromptModel {
	return &stashPromptModel{
		choices: []string{
			"Stash changes and continue",
			"Abort checkout",
		},
		cursor:   0,
		selected: false,
	}
}

// Init initializes the model
func (m *stashPromptModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m *stashPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = true
			return m, tea.Quit
		case "q", "esc":
			m.cursor = 1 // Select "Abort checkout"
			m.selected = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the model
func (m *stashPromptModel) View() string {
	if m.selected {
		return ""
	}

	s := "You have uncommitted changes that would be overwritten.\n"
	s += "What would you like to do?\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	s += "\nPress Enter to confirm, q or Esc to abort"
	return s
}
