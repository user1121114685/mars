package cmd

import (
	"mars/internal/common/version"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "查看版本号",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println(version.Format())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
