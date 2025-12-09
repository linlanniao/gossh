package cmd

import (
	"fmt"

	"gossh/pkg/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "显示版本信息",
	Aliases: []string{"v"},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Print("gossh"))
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

