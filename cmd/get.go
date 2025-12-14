package cmd

import (
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:                "get",
	Short:              "Run kubectl get against all contexts",
	Long:               `Run kubectl get command against all contexts in parallel.`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCommand("get", args)
	},
}
