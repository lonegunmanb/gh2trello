package cmd

import (
	"fmt"
	"strconv"
	"strings"

	gh "github.com/lonegunmanb/gh2trello/github"
	"github.com/lonegunmanb/gh2trello/sync"
	"github.com/lonegunmanb/gh2trello/trello"
	"github.com/spf13/cobra"
)

var workitemCmd = &cobra.Command{
	Use:   "workitem <owner/repo> <number>",
	Short: "Sync a single GitHub issue or PR to Trello",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoFullName := args[0]
		parts := strings.SplitN(repoFullName, "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid repository format: %s (expected owner/repo)", repoFullName)
		}
		owner, repo := parts[0], parts[1]

		number, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid issue/PR number: %s", args[1])
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		if cfg.GitHubToken == "" {
			return fmt.Errorf("github token is required (--github-token or GITHUB_TOKEN)")
		}
		if cfg.TrelloAPIKey == "" || cfg.TrelloAPIToken == "" {
			return fmt.Errorf("trello API key and token are required")
		}
		if cfg.TrelloIssueListID == "" || cfg.TrelloPRListID == "" {
			return fmt.Errorf("trello issue list ID and PR list ID are required")
		}

		ghClient := gh.NewClient(cfg.GitHubToken)
		trelloClient := trello.NewClient(cfg.TrelloAPIKey, cfg.TrelloAPIToken)

		engine := sync.NewEngine(ghClient, trelloClient, cfg, cfgPath)

		if err := engine.SyncWorkitem(cmd.Context(), owner, repo, number); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Synced %s/%s#%d to Trello\n", owner, repo, number)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(workitemCmd)
}
