package cli

import (
	"encoding/json"
	"os"
	"path/filepath"

	"glyph/internal/store"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "glyph",
		Short:         "Agent-native source control",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate("glyph {{.Version}}\n")
	root.PersistentFlags().Bool("json", false, "emit stable JSON output for agents")
	root.AddCommand(
		versionCmd(),
		installCmd(),
		initCmd(),
		importCmd(),
		statusCmd(),
		graphCmd(),
		workCmd(),
		readCmd(),
		writeCmd(),
		projectCmd(),
		diffCmd(),
		checkpointCmd(),
		publishCmd(),
		publicationCmd(),
		hookCmd(),
		docsCmd(),
		skillsCmd(),
		vizCmd(),
		exportCmd(),
		remoteCmd(),
		mountCmd(),
	)
	return root
}

func Execute() error {
	cmd := NewRootCommand()
	if err := cmd.Execute(); err != nil {
		if modeFrom(cmd).JSON {
			_ = writeJSONError(cmd, err)
			return jsonReportedError{err: err}
		}
		return err
	}
	return nil
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show Glyph version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := versionInfo{
				Version: version,
				Commit:  commit,
				Date:    date,
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "version", info)
			}
			return humanf(cmd, "glyph %s\ncommit: %s\nbuilt: %s\n", info.Version, info.Commit, info.Date)
		},
	}
}

type jsonReportedError struct {
	err error
}

func (e jsonReportedError) Error() string {
	return e.err.Error()
}

func (e jsonReportedError) JSONReported() bool {
	return true
}

func (e jsonReportedError) Unwrap() error {
	return e.err
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a Glyph store",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			st, err := store.Init(cwd)
			if err != nil {
				return err
			}
			defer st.Close()
			storePath := filepath.Join(cwd, store.DirName)
			if err := humanf(cmd, "initialized %s\n", storePath); err != nil {
				return err
			}
			return writeResponse(cmd, "init", map[string]any{"store": storePath})
		},
	}
}

func importCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import [path]",
		Short: "Import the bootstrap workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			count, err := st.ImportWorkspace()
			if err != nil {
				return err
			}
			if err := humanf(cmd, "imported %d files\n", count); err != nil {
				return err
			}
			return writeResponse(cmd, "import", map[string]any{"files": count})
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Glyph store status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			counts, err := st.Counts()
			if err != nil {
				return err
			}
			data := map[string]any{
				"store":      st.Dir,
				"database":   filepath.Join(st.Dir, "store.db"),
				"store_role": "local source-control database",
				"version":    1,
				"counts":     counts,
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "status", data)
			}
			if err := humanf(cmd, "store: %s\n", st.Dir); err != nil {
				return err
			}
			if err := humanf(cmd, "database: %s\n", filepath.Join(st.Dir, "store.db")); err != nil {
				return err
			}
			if err := humanf(cmd, "store_role: local source-control database\n"); err != nil {
				return err
			}
			if err := humanf(cmd, "version: 1\n"); err != nil {
				return err
			}
			for _, key := range []string{"sources", "content", "realms", "work_contexts", "snapshots", "publications", "remotes", "mounts", "work_claims", "work_dependencies", "work_conflicts", "hook_runs"} {
				if err := humanf(cmd, "%s: %d\n", key, counts[key]); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func graphCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "graph",
		Short: "Show source graph summary",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			counts, err := st.Counts()
			if err != nil {
				return err
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "graph", counts)
			}
			b, _ := json.MarshalIndent(counts, "", "  ")
			return humanf(cmd, "%s\n", string(b))
		},
	}
}
