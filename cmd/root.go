package cmd

import (
	"github.com/ducconit/tools/cmd/vue"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tools",
	Short: "Tools for dev",
	Long:  `Tools for dev`,
}

func init() {
	rootCmd.AddCommand(vue.UpdateI18nDefaultValueCmd)
	rootCmd.AddCommand(vue.I18nScanCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
