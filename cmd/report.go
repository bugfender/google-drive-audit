package cmd

import (
	"context"
	"google-drive-audit/audit"
	"google-drive-audit/util"
	"os"

	"github.com/spf13/cobra"
)

var (
	output string

	reportCmd = &cobra.Command{
		Use:     "report",
		Short:   "List all files with permissions",
		Long:    `report lists all files for all users in all the company's drives, together with who has access to them.`,
		Example: "google-drive-audit report",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			out := os.Stdout
			if output != "-" {
				var err error
				out, err = os.Create(output)
				if err != nil {
					return err
				}
				defer util.PrintIfError(out.Close)
			}

			return audit.WriteFileReport(ctx, database, out)
		},
	}
)

func init() {
	reportCmd.PersistentFlags().StringVarP(&database, "database", "b", "db.json", "database file")
	reportCmd.PersistentFlags().StringVarP(&output, "output", "o", "-", "output file")
}
