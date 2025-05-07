package git

import (
	"sort"
	"strconv"
	"strings"
)

// branchMatch represents a branch that matches the search pattern
type branchMatch struct {
	name    string
	isLocal bool
	score   int
}

// calcMatchScore calculates how well a branch matches the pattern
// Higher scores are better matches
func calcMatchScore(branch, pattern string) int {
	branchLower := strings.ToLower(branch)
	patternLower := strings.ToLower(pattern)

	score := 0

	// Check for exact match - highest priority
	if branchLower == patternLower {
		return 10000
	}

	// Check if pattern is a number (like "123")
	if num, err := strconv.Atoi(pattern); err == nil {
		// If branch contains the ticket number
		if strings.Contains(branch, "#"+pattern) {
			return 600
		}
		// If branch contains the number anywhere
		if strings.Contains(branch, strconv.Itoa(num)) {
			return 400
		}
	}

	// Check if branch ends with pattern
	if strings.HasSuffix(branchLower, patternLower) {
		score += 1000
	}

	// Check if branch starts with pattern
	if strings.HasPrefix(branchLower, patternLower) {
		score += 500
	}

	// Check if branch contains pattern as a whole word
	if strings.Contains(branchLower, "/"+patternLower+"/") ||
		strings.Contains(branchLower, "/"+patternLower) ||
		strings.Contains(branchLower, patternLower+"/") {
		score += 300
	}

	// Check if branch contains all characters of pattern in order (even with gaps)
	if containsSubsequence(branchLower, patternLower) {
		score += 250
	}

	// Check if branch contains pattern
	if strings.Contains(branchLower, patternLower) {
		score += 100
	}

	// Penalty for longer branch names
	score -= len(branch) / 5

	// Favor common branch names
	commonBranches := map[string]int{
		"master":     50,
		"main":       50,
		"develop":    40,
		"dev":        40,
		"production": 40,
		"prod":       40,
		"staging":    30,
		"stage":      30,
		"test":       20,
	}

	// Add score for common branch names
	for commonBranch, bonus := range commonBranches {
		if branchLower == commonBranch && strings.Contains(commonBranch, patternLower) {
			score += bonus
		}
	}

	return score
}

// containsSubsequence checks if a string contains all characters of a subsequence in order
// For example, "chestag" is a subsequence of "cheddar/staging"
func containsSubsequence(s, subseq string) bool {
	if len(subseq) == 0 {
		return true
	}

	// Find each character of subseq in order
	idx := 0
	for i := 0; i < len(s) && idx < len(subseq); i++ {
		if s[i] == subseq[idx] {
			idx++
		}
	}

	return idx == len(subseq)
}

// sortMatches sorts branch matches by score (higher is better)
func sortMatches(matches []branchMatch) {
	sort.Slice(matches, func(i, j int) bool {
		// If scores are equal, prioritize local branches
		if matches[i].score == matches[j].score {
			return matches[i].isLocal && !matches[j].isLocal
		}
		return matches[i].score > matches[j].score
	})
}

// createFilteredBranchModel creates a branch model with only matching branches
func createFilteredBranchModel(matches []branchMatch, debugMode bool) branchModel {
	// Convert branch matches to Branch objects
	branches := make([]Branch, len(matches))
	for i, match := range matches {
		branches[i] = Branch{
			Name:    match.name,
			IsLocal: match.isLocal,
			Current: false, // We'll set this later
		}
	}

	// Get current branch to mark it
	currentBranch, err := getCurrentBranch()
	if err == nil {
		for i, branch := range branches {
			if branch.Name == currentBranch {
				branches[i].Current = true
				break
			}
		}
	}

	// Create model with filtered branches
	model := branchModel{
		branches:    branches,
		selected:    0,
		query:       "",
		width:       80,
		height:      20,
		showRemotes: true,
		debugMode:   debugMode,
	}

	// Initial filter to show all matches
	model.filteredIdx = make([]int, len(branches))
	for i := range branches {
		model.filteredIdx[i] = i
	}

	return model
}
