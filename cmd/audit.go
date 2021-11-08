package cmd

import (
	"context"
	"google-drive-audit/audit"
	"google-drive-audit/util"
	"os"

	"github.com/spf13/cobra"
)

var (
	domain          string
	adminEmail      string
	output          string
	quiet           bool
	credentialsFile string

	auditCmd = &cobra.Command{
		Use:     "audit",
		Short:   "List all files with permissions",
		Long:    `audit lists all files for all users in all the company's drives, together with who has access to them.`,
		Example: "google-drive-audit audit --domain=yourcompany.com --admin-email=you@yourcompany.com",
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

			return audit.ExportFileListToCSV(ctx, out, domain, adminEmail, false, !quiet, credentialsFile)
		},
	}
)

func init() {
	auditCmd.PersistentFlags().StringVarP(&domain, "domain", "d", "", "domain name to audit")
	if err := auditCmd.MarkPersistentFlagRequired("domain"); err != nil {
		panic(err)
	}
	auditCmd.PersistentFlags().StringVarP(&adminEmail, "admin-email", "a", "", "email address of a domain administrator")
	if err := auditCmd.MarkPersistentFlagRequired("admin-email"); err != nil {
		panic(err)
	}
	auditCmd.PersistentFlags().StringVarP(&output, "output", "o", "-", "output file")
	auditCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "do not show progress")
	auditCmd.PersistentFlags().StringVarP(&credentialsFile, "credentials", "c", "credentials.json", "service credentials file (obtained from Google Cloud Platform console)")
}
