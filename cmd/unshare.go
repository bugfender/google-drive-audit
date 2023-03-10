package cmd

import (
	"context"
	"google-drive-audit/audit"
	"google-drive-audit/util"
	"os"

	"github.com/spf13/cobra"
)

var (
	email  string
	dryRun bool

	unshareCmd = &cobra.Command{
		Use:     "unshare",
		Short:   "Removes all permissions for a given user",
		Long:    `unshare finds all files shared with a given user in all the company's drives, and removes this user's permissions.`,
		Example: "google-drive-audit unshare --domain yourcompany.com --admin-email you@yourcompany.com --user ex.employee@yourcompany.com",
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

			return audit.UnshareFiles(ctx, domain, adminEmail, credentialsFile, database, email, dryRun)
		},
	}
)

func init() {
	unshareCmd.PersistentFlags().StringVarP(&domain, "domain", "d", "", "domain name to audit")
	if err := unshareCmd.MarkPersistentFlagRequired("domain"); err != nil {
		panic(err)
	}
	unshareCmd.PersistentFlags().StringVarP(&adminEmail, "admin-email", "a", "", "email address of a domain administrator")
	if err := unshareCmd.MarkPersistentFlagRequired("admin-email"); err != nil {
		panic(err)
	}
	unshareCmd.PersistentFlags().StringVarP(&database, "database", "b", "db.json", "database file")
	unshareCmd.PersistentFlags().StringVarP(&credentialsFile, "credentials", "c", "credentials.json", "service credentials file (obtained from Google Cloud Platform console)")
	unshareCmd.PersistentFlags().StringVarP(&email, "user", "u", "-", "user email to remove permissions from")
	unshareCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n", false, "print actions instead of performing them")
}
