package cmd

import (
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:                "version",
	Short:              "Run kubectl version against all contexts",
	Long:               `Run kubectl version command against all contexts in parallel.`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCommand("version", args)
	},
}
