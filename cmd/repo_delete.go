package cmd

import (
	"fmt"

	"github.com/lonegunmanb/gh2trello/config"
	"github.com/spf13/cobra"
)

var repoDeleteCmd = &cobra.Command{
	Use:   "delete <owner/repo>",
	Short: "Remove a repository from monitoring",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoName := args[0]

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		cfg.DeleteRepo(repoName)

		if err := config.Save(cfgPath, cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Deleted repository: %s\n", repoName)
		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoDeleteCmd)
}
