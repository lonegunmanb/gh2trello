package cmd

import (
	"fmt"
	"os"

	"github.com/lonegunmanb/gh2trello/config"
	"github.com/spf13/cobra"
)

var (
	cfgPath         string
	githubToken     string
	trelloAPIKey    string
	trelloAPIToken  string
	trelloBoardID   string
	trelloIssueList string
	trelloPRList    string
)

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "gh2trello",
	Short: "Sync GitHub issues and PRs to Trello boards",
	Long:  "gh2trello is a CLI tool that synchronizes GitHub repository issues and pull requests to Trello boards.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "gh2trello.json", "config file path")
	rootCmd.PersistentFlags().StringVar(&githubToken, "github-token", "", "GitHub personal access token")
	rootCmd.PersistentFlags().StringVar(&trelloAPIKey, "trello-api-key", "", "Trello API key")
	rootCmd.PersistentFlags().StringVar(&trelloAPIToken, "trello-api-token", "", "Trello API token")
	rootCmd.PersistentFlags().StringVar(&trelloBoardID, "trello-board-id", "", "Trello board ID")
	rootCmd.PersistentFlags().StringVar(&trelloIssueList, "trello-issue-list-id", "", "Trello issue list ID")
	rootCmd.PersistentFlags().StringVar(&trelloPRList, "trello-pr-list-id", "", "Trello PR list ID")
}

// loadConfig loads the config and applies flag/env overrides.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Flag > config > env
	applyOverride(&cfg.GitHubToken, githubToken, "GITHUB_TOKEN")
	applyOverride(&cfg.TrelloAPIKey, trelloAPIKey, "TRELLO_API_KEY")
	applyOverride(&cfg.TrelloAPIToken, trelloAPIToken, "TRELLO_API_TOKEN")
	applyOverride(&cfg.TrelloBoardID, trelloBoardID, "TRELLO_BOARD_ID")
	applyOverride(&cfg.TrelloIssueListID, trelloIssueList, "TRELLO_ISSUE_LIST_ID")
	applyOverride(&cfg.TrelloPRListID, trelloPRList, "TRELLO_PR_LIST_ID")

	return cfg, nil
}

// applyOverride sets *target to flagVal if non-empty, else envKey if set.
func applyOverride(target *string, flagVal, envKey string) {
	if flagVal != "" {
		*target = flagVal
		return
	}
	if *target == "" {
		if v := os.Getenv(envKey); v != "" {
			*target = v
		}
	}
}
