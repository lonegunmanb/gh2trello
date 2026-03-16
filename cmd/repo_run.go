package cmd

import (
	"fmt"

	gh "github.com/lonegunmanb/gh2trello/github"
	"github.com/lonegunmanb/gh2trello/sync"
	"github.com/lonegunmanb/gh2trello/trello"
	"github.com/spf13/cobra"
)

var pollInterval string

var repoRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Start polling and syncing all configured repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		// Override poll interval if flag is set
		if pollInterval != "" {
			cfg.PollInterval = pollInterval
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

		fmt.Fprintf(cmd.OutOrStdout(), "Starting polling with interval %s...\n", cfg.PollInterval)
		return engine.RunPolling(cmd.Context())
	},
}

func init() {
	repoRunCmd.Flags().StringVar(&pollInterval, "poll-interval", "", "polling interval (e.g. 5m, 30s)")
	repoCmd.AddCommand(repoRunCmd)
}
