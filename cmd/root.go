package cmd

import (
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "google-drive-audit",
		Short: "A utility to assist in the audit of Google Drive",
		Long:  `google-drive-audit lists all files for all users in all the company's drives, together with who has access to them.`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(unshareCmd)
}
