package cli

import (
	"fmt"
	"strings"
	"time"

	"glyph/internal/store"

	"github.com/spf13/cobra"
)

func checkCmd() *cobra.Command {
	var keep bool
	var out string
	var names []string
	cmd := &cobra.Command{
		Use:   "check <realm>",
		Short: "Run export checks for a realm",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			defaults, err := store.LoadBootstrapManifest(st.Root)
			if err != nil {
				return err
			}
			result, err := st.CheckRealmExport(store.CheckOptions{
				Realm: args[0],
				Out:   out,
				Keep:  keep,
				Names: names,
				GitOptions: store.GitExportOptions{
					Gitignore:  defaults.Defaults.Export.Git.Gitignore,
					Gitinclude: defaults.Defaults.Export.Git.Gitinclude,
				},
				Timeout: 2 * time.Minute,
			})
			if modeFrom(cmd).JSON {
				if result != nil {
					_ = writeResponse(cmd, "check", result)
				}
				return err
			}
			if result != nil {
				for _, check := range result.Checks {
					status := "ok"
					if !check.OK {
						status = "failed"
					}
					if humanErr := humanf(cmd, "%s\t%s\t%s\n", status, check.Name, check.Command); humanErr != nil {
						return humanErr
					}
				}
			}
			if err != nil {
				return err
			}
			if result == nil || len(result.Checks) == 0 {
				return humanf(cmd, "no checks configured for %s\n", args[0])
			}
			return humanf(cmd, "checks passed for %s\n", args[0])
		},
	}
	cmd.Flags().BoolVar(&keep, "keep", false, "keep the temporary exported check directory")
	cmd.Flags().StringVar(&out, "out", "", "directory to use for the check export")
	cmd.Flags().StringArrayVar(&names, "check", nil, "run only the named check; repeat for multiple checks")
	return cmd
}

func runPublicExportChecks(st *store.Store, defaults *store.BootstrapManifest, names []string) (*store.CheckResult, error) {
	return st.CheckRealmExport(store.CheckOptions{
		Realm: "public",
		Names: names,
		GitOptions: store.GitExportOptions{
			Gitignore:  defaults.Defaults.Export.Git.Gitignore,
			Gitinclude: defaults.Defaults.Export.Git.Gitinclude,
		},
		Timeout: 2 * time.Minute,
	})
}

func formatCheckFailure(result *store.CheckResult, err error) error {
	if result == nil {
		return err
	}
	var failed []string
	for _, check := range result.Checks {
		if !check.OK {
			failed = append(failed, check.Name)
		}
	}
	if len(failed) == 0 {
		return err
	}
	return fmt.Errorf("%w: %s", err, strings.Join(failed, ", "))
}
