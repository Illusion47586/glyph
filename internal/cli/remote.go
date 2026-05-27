package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"glyph/internal/store"

	"github.com/spf13/cobra"
)

func exportCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "export", Short: "Export projections"}
	var realm, out, gitignore, gitinclude string
	gitCmd := &cobra.Command{
		Use:   "git",
		Short: "Export a realm to a clean Git repository",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if out == "" {
				return fmt.Errorf("--out is required")
			}
			defaults, err := commandDefaults()
			if err != nil {
				return err
			}
			if !flagChanged(cmd, "gitignore") && defaults.Defaults.Export.Git.Gitignore != "" {
				gitignore = defaults.Defaults.Export.Git.Gitignore
			}
			if !flagChanged(cmd, "gitinclude") && defaults.Defaults.Export.Git.Gitinclude != "" {
				gitinclude = defaults.Defaults.Export.Git.Gitinclude
			}
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			result, err := st.ExportGitWithOptions(realm, out, store.GitExportOptions{Gitignore: gitignore, Gitinclude: gitinclude})
			if err != nil {
				return err
			}
			if err := humanf(cmd, "exported %s to %s\n", realm, out); err != nil {
				return err
			}
			return writeResponse(cmd, "git_export", result)
		},
	}
	gitCmd.Flags().StringVar(&realm, "realm", "public", "realm to export")
	gitCmd.Flags().StringVar(&out, "out", "", "output directory")
	gitCmd.Flags().StringVar(&gitignore, "gitignore", store.GitCompatNone, "generate .gitignore: none, generated, or overwrite")
	gitCmd.Flags().StringVar(&gitinclude, "gitinclude", store.GitCompatNone, "generate .gitinclude: none, generated, or overwrite")
	cmd.AddCommand(gitCmd)
	return cmd
}

func remoteCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "remote", Short: "Manage remotes"}
	var mode string
	addCmd := &cobra.Command{
		Use:   "add <name> <spec>",
		Short: "Add a remote",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			if err := st.AddRemote(args[0], args[1], mode); err != nil {
				return err
			}
			if err := humanf(cmd, "added remote %s\n", args[0]); err != nil {
				return err
			}
			return writeResponse(cmd, "remote_added", map[string]any{"name": args[0], "spec": args[1], "mode": mode})
		},
	}
	addCmd.Flags().StringVar(&mode, "mode", "export-only", "remote mode")
	cmd.AddCommand(addCmd, remoteListCmd(), remoteInspectCmd(), remoteSyncCmd())
	return cmd
}

func remoteListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List remotes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			remotes, err := st.Remotes()
			if err != nil {
				return err
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "remote_list", remotes)
			}
			for _, remote := range remotes {
				if err := humanf(cmd, "%s\t%s\t%s\n", remote["name"], remote["spec"], remote["mode"]); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func remoteInspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <name>",
		Short: "Inspect a remote",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			remote, err := st.Remote(args[0])
			if err != nil {
				return err
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "remote", remote)
			}
			b, _ := json.MarshalIndent(remote, "", "  ")
			return humanf(cmd, "%s\n", string(b))
		},
	}
}

func remoteSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync <name>",
		Short: "Sync a remote",
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
			result, err := st.SyncRemoteWithOptions(args[0], store.GitExportOptions{
				Gitignore:  defaults.Defaults.Export.Git.Gitignore,
				Gitinclude: defaults.Defaults.Export.Git.Gitinclude,
			})
			if err != nil {
				return err
			}
			if err := humanf(cmd, "synced %s\n", args[0]); err != nil {
				return err
			}
			return writeResponse(cmd, "remote_synced", result)
		},
	}
}

func mountCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "mount", Short: "Manage external source mounts"}
	cmd.AddCommand(&cobra.Command{
		Use:   "add <path> <spec>",
		Short: "Mounts are recorded in later prototypes",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("mount add is specified but deferred from prototype 0")
		},
	})
	cmd.AddCommand(&cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("mount list is specified but deferred from prototype 0")
	}})
	cmd.AddCommand(&cobra.Command{Use: "update <path>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("mount update is specified but deferred from prototype 0")
	}})
	cmd.AddCommand(&cobra.Command{Use: "inspect <path>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("mount inspect is specified but deferred from prototype 0")
	}})
	return cmd
}

func openStore() (*store.Store, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return store.Open(cwd)
}
