package cmd

import (
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "stats",
	Long:  "stats",
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
