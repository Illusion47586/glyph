package cli

import (
	"encoding/base64"
	"io"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

func readCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read <work> <path>",
		Short: "Read a file from a work context",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			data, err := st.ReadWorkFile(args[0], args[1])
			if err != nil {
				return err
			}
			if modeFrom(cmd).JSON {
				payload := map[string]any{
					"work":     args[0],
					"path":     args[1],
					"encoding": "utf-8",
					"content":  string(data),
				}
				if !utf8.Valid(data) {
					payload["encoding"] = "base64"
					payload["content"] = base64.StdEncoding.EncodeToString(data)
				}
				return writeResponse(cmd, "read", payload)
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
}

func writeCmd() *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "write <work> <path>",
		Short: "Write stdin to a file in a work context",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if reason == "" {
				reason = "cli write"
			}
			data, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return err
			}
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			if err := st.WriteWorkFile(args[0], args[1], data, reason); err != nil {
				return err
			}
			if err := humanf(cmd, "wrote %s in %s\n", args[1], args[0]); err != nil {
				return err
			}
			return writeResponse(cmd, "write", map[string]any{"work": args[0], "path": args[1], "reason": reason})
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "write reason for provenance")
	return cmd
}

func projectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "project <work> <dir>",
		Short: "Materialize a work context into a directory",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			if err := st.ProjectWork(args[0], args[1]); err != nil {
				return err
			}
			if err := humanf(cmd, "projected %s to %s\n", args[0], args[1]); err != nil {
				return err
			}
			return writeResponse(cmd, "project", map[string]any{"work": args[0], "path": args[1]})
		},
	}
}

func diffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <work>",
		Short: "Show file-level work context diff",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			changes, err := st.DiffWork(args[0])
			if err != nil {
				return err
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "diff", map[string]any{"work": args[0], "changes": changes})
			}
			for _, change := range changes {
				if err := humanf(cmd, "%s\n", change); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
