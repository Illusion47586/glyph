package cli

import (
	"fmt"

	"glyph/internal/store"

	"github.com/spf13/cobra"
)

func checkpointCmd() *cobra.Command {
	var message string
	cmd := &cobra.Command{
		Use:   "checkpoint <work>",
		Short: "Create an explicit milestone snapshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if message == "" {
				message = "checkpoint"
			}
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			if err := st.Snapshot(args[0], message); err != nil {
				return err
			}
			if err := humanf(cmd, "checkpointed %s\n", args[0]); err != nil {
				return err
			}
			return writeResponse(cmd, "checkpoint", map[string]any{"work": args[0], "message": message})
		},
	}
	cmd.Flags().StringVar(&message, "message", "", "checkpoint message")
	return cmd
}

func publishCmd() *cobra.Command {
	var to, mode, semanticType, semanticScope, semanticDescription string
	cmd := &cobra.Command{
		Use:   "publish <work>",
		Short: "Publish a work context into a realm",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			id, err := st.PublishWithOptions(store.PublishOptions{
				Work:                args[0],
				DestRealm:           to,
				Mode:                mode,
				SemanticType:        semanticType,
				SemanticScope:       semanticScope,
				SemanticDescription: semanticDescription,
			})
			if err != nil {
				return err
			}
			if err := humanf(cmd, "published %s\n", id); err != nil {
				return err
			}
			data := map[string]any{"publication": id, "work": args[0], "realm": to, "mode": mode}
			if semanticType != "" {
				data["semantic_type"] = semanticType
				data["semantic_scope"] = semanticScope
				data["semantic_description"] = semanticDescription
			}
			return writeResponse(cmd, "publish", data)
		},
	}
	cmd.Flags().StringVar(&to, "to", "public", "destination realm")
	cmd.Flags().StringVar(&mode, "mode", "squash", "publication history mode: squash or preserve")
	cmd.Flags().StringVar(&semanticType, "semantic-type", "", "semantic commit type for exported Git commits, such as feat, fix, docs, chore, or ci")
	cmd.Flags().StringVar(&semanticScope, "semantic-scope", "", "optional semantic commit scope for exported Git commits")
	cmd.Flags().StringVar(&semanticDescription, "semantic-description", "", "semantic commit description for exported Git commits")
	return cmd
}

func publicationCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "publication", Short: "Manage publications"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List publications",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			pubs, err := st.ListPublications()
			if err != nil {
				return err
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "publication_list", pubs)
			}
			for _, pub := range pubs {
				if err := humanf(cmd, "%s\t%s\t%s\t%s\n", pub["id"], pub["work"], pub["realm"], pub["status"]); err != nil {
					return err
				}
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "inspect <id>",
		Short: "Inspect publication metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("publication inspect is not implemented in prototype 0")
		},
	})
	return cmd
}

func hookCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "hook", Short: "Inspect and run local hooks"}
	cmd.AddCommand(hookListCmd(), hookRunCmd())
	return cmd
}

func hookListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed local hooks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			hooks, err := st.ListHooks()
			if err != nil {
				return err
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "hook_list", hooks)
			}
			for _, hook := range hooks {
				if err := humanf(cmd, "%s\t%s\texecutable=%t\n", hook.Event, hook.Path, hook.Executable); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func hookRunCmd() *cobra.Command {
	var work, to, mode string
	cmd := &cobra.Command{
		Use:   "run <event>",
		Short: "Run a local hook event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			run, err := st.RunHook(store.HookContext{Event: args[0], Work: work, DestRealm: to, Mode: mode, Actor: store.BootstrapUser})
			if err != nil {
				return err
			}
			if run == nil {
				if err := humanf(cmd, "hook %s not installed\n", args[0]); err != nil {
					return err
				}
				return writeResponse(cmd, "hook_run", map[string]any{"event": args[0], "installed": false})
			}
			if err := humanf(cmd, "hook %s exited %d\n", run.Event, run.ExitCode); err != nil {
				return err
			}
			return writeResponse(cmd, "hook_run", run)
		},
	}
	cmd.Flags().StringVar(&work, "work", "", "work context for hook context")
	cmd.Flags().StringVar(&to, "to", "public", "destination realm for hook context")
	cmd.Flags().StringVar(&mode, "mode", "squash", "publication mode for hook context")
	return cmd
}
