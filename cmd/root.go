package cmd

import (
	"fmt"
	"os"

	"github.com/reckerp/gch/git"
	"github.com/spf13/cobra"
)

var (
	debugMode    bool
	createBranch bool
	force        bool
	stash        bool

	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "gch [pattern]",
		Short: "Smart Git branch checkout tool with fuzzy matching",
		Long: `gch is an intelligent Git branch checkout tool that provides fast and intuitive branch switching.
It uses fuzzy matching to find branches based on partial names, making it easy to switch between branches
without typing the full name.

Features:
  • Fuzzy branch name matching
  • Interactive branch selector
  • Remote branch tracking
  • Smart branch creation
  • Force checkout support
  • Automatic stashing

Examples:
  # Checkout a branch using partial name
  gch prod            # Checkout branch containing 'prod'
  gch 123             # Checkout branch containing '123'
  
  # Create and checkout a new branch
  gch -b feature      # Create and checkout new branch 'feature'
  gch -b feat/user    # Create and checkout new branch 'feat/user'
  
  # Force checkout (discard local changes)
  gch -f prod         # Force checkout branch containing 'prod'
  gch -b -f feature   # Force create and checkout new branch
  
  # Always stash changes before checkout
  gch -s prod         # Stash changes and checkout branch containing 'prod'
  gch -s -b feature   # Stash changes and create/checkout new branch
  
  # Show interactive branch selector
  gch                 # List all branches for interactive selection`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Check if we're in a git repository
			if !git.IsGitRepo() {
				fmt.Fprintln(os.Stderr, "Error: not a git repository")
				os.Exit(1)
			}

			pattern := ""
			if len(args) > 0 {
				pattern = args[0]
			}

			// If no pattern provided, show interactive branch selector
			if pattern == "" {
				if err := git.ShowInteractiveBranchSelector(debugMode); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				return
			}

			// Otherwise use smart checkout with pattern
			err := git.SmartCheckout(pattern, createBranch, force, stash, debugMode)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}
)

func init() {
	RootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug output for branch matching process")
	RootCmd.Flags().BoolVarP(&createBranch, "branch", "b", false, "Create and checkout a new branch with the given name")
	RootCmd.Flags().BoolVarP(&force, "force", "f", false, "Force checkout, discarding any local changes")
	RootCmd.Flags().BoolVarP(&stash, "stash", "s", false, "Always stash changes before checkout")
}
