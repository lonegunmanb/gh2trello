package cmd

import (
	"fmt"

	"github.com/lonegunmanb/gh2trello/config"
	"github.com/spf13/cobra"
)

var (
	repoAddSince string
	repoAddQuery string
)

var repoAddCmd = &cobra.Command{
	Use:   "add <owner/repo>",
	Short: "Add a repository to monitor",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoName := args[0]

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		cfg.AddRepo(repoName, repoAddSince, repoAddQuery)

		if err := config.Save(cfgPath, cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Added repository: %s\n", repoName)
		return nil
	},
}

func init() {
	repoAddCmd.Flags().StringVar(&repoAddSince, "since", "", "watermark time (RFC3339), empty means now")
	repoAddCmd.Flags().StringVar(&repoAddQuery, "query", "", "GitHub search query filter")
	repoCmd.AddCommand(repoAddCmd)
}
