package cli

import (
	"fmt"
	"time"

	"glyph/internal/store"

	"github.com/spf13/cobra"
)

func workCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "work", Short: "Manage work contexts"}
	cmd.AddCommand(
		workStartCmd(),
		workListCmd(),
		workStatusCmd(),
		workSnapshotCmd(),
		workClaimCmd(),
		workHeartbeatCmd(),
		workReleaseCmd(),
		workConflictsCmd(),
		workDependCmd(),
		workPruneCmd(),
		workDiscardCmd(),
	)
	return cmd
}

func workStartCmd() *cobra.Command {
	var from string
	cmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Start a work context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			wc, err := st.StartWork(args[0], from)
			if err != nil {
				return err
			}
			data := map[string]any{"name": wc.Name, "base_realm": wc.BaseRealm, "workspace": wc.WorkspacePath, "status": wc.Status}
			if err := humanf(cmd, "started %s from %s\nworkspace: %s\n", wc.Name, wc.BaseRealm, wc.WorkspacePath); err != nil {
				return err
			}
			return writeResponse(cmd, "work_started", data)
		},
	}
	cmd.Flags().StringVar(&from, "from", "public", "realm projection to start from")
	return cmd
}

func workListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List work contexts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			work, err := st.ListWork()
			if err != nil {
				return err
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "work_list", work)
			}
			for _, wc := range work {
				if err := humanf(cmd, "%s\t%s\t%s\n", wc.Name, wc.BaseRealm, wc.Status); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func workStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <name>",
		Short: "Show work context status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			wc, err := st.Work(args[0])
			if err != nil {
				return err
			}
			data := map[string]any{"name": wc.Name, "base_realm": wc.BaseRealm, "status": wc.Status, "workspace": wc.WorkspacePath}
			if err := humanf(cmd, "name: %s\nbase: %s\nstatus: %s\nworkspace: %s\n", wc.Name, wc.BaseRealm, wc.Status, wc.WorkspacePath); err != nil {
				return err
			}
			return writeResponse(cmd, "work_status", data)
		},
	}
}

func workSnapshotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "snapshot <name>",
		Short: "Snapshot a work context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			if err := st.Snapshot(args[0], "manual snapshot"); err != nil {
				return err
			}
			if err := humanf(cmd, "snapshotted %s\n", args[0]); err != nil {
				return err
			}
			return writeResponse(cmd, "snapshot", map[string]any{"work": args[0]})
		},
	}
}

func workClaimCmd() *cobra.Command {
	var actor, mode, ttl string
	cmd := &cobra.Command{
		Use:   "claim <name>",
		Short: "Claim a work context for an actor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := parseDurationFlag(ttl)
			if err != nil {
				return err
			}
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			claim, err := st.ClaimWork(args[0], actor, mode, d)
			if err != nil {
				return err
			}
			if err := humanf(cmd, "claimed %s for %s until %s\n", claim.WorkName, claim.Actor, claim.ExpiresAt); err != nil {
				return err
			}
			return writeResponse(cmd, "work_claimed", claim)
		},
	}
	cmd.Flags().StringVar(&actor, "actor", store.BootstrapUser, "actor identity claiming the work context")
	cmd.Flags().StringVar(&mode, "mode", "exclusive", "claim mode: exclusive, shared-read, or handoff")
	cmd.Flags().StringVar(&ttl, "ttl", "15m", "claim time-to-live")
	return cmd
}

func workHeartbeatCmd() *cobra.Command {
	var actor, ttl string
	cmd := &cobra.Command{
		Use:   "heartbeat <name>",
		Short: "Refresh a work context claim heartbeat",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := parseDurationFlag(ttl)
			if err != nil {
				return err
			}
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			claim, err := st.HeartbeatWork(args[0], actor, d)
			if err != nil {
				return err
			}
			if err := humanf(cmd, "heartbeat %s for %s until %s\n", claim.WorkName, claim.Actor, claim.ExpiresAt); err != nil {
				return err
			}
			return writeResponse(cmd, "work_heartbeat", claim)
		},
	}
	cmd.Flags().StringVar(&actor, "actor", store.BootstrapUser, "actor identity refreshing the heartbeat")
	cmd.Flags().StringVar(&ttl, "ttl", "15m", "claim time-to-live")
	return cmd
}

func workReleaseCmd() *cobra.Command {
	var actor string
	cmd := &cobra.Command{
		Use:   "release <name>",
		Short: "Release a work context claim",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			if err := st.ReleaseWork(args[0], actor); err != nil {
				return err
			}
			if err := humanf(cmd, "released %s for %s\n", args[0], actor); err != nil {
				return err
			}
			return writeResponse(cmd, "work_released", map[string]any{"work": args[0], "actor": actor})
		},
	}
	cmd.Flags().StringVar(&actor, "actor", store.BootstrapUser, "actor identity releasing the claim")
	return cmd
}

func workConflictsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "conflicts <name>",
		Short: "Detect conflicts for a work context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			conflicts, err := st.WorkConflicts(args[0])
			if err != nil {
				return err
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "work_conflicts", map[string]any{"work": args[0], "conflicts": conflicts})
			}
			for _, conflict := range conflicts {
				if err := humanf(cmd, "%s\t%s\t%s\t%s\n", conflict.Path, conflict.Type, conflict.OtherWork, conflict.Detail); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func workDependCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "depend <work> <depends-on>",
		Short: "Record work context dependency ordering",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			dep, err := st.AddDependency(args[0], args[1])
			if err != nil {
				return err
			}
			if err := humanf(cmd, "%s depends on %s\n", dep.WorkName, dep.DependsOnWork); err != nil {
				return err
			}
			return writeResponse(cmd, "work_dependency", dep)
		},
	}
}

func workPruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prune <name>",
		Short: "Prune a completed work context projection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			if err := st.PruneWork(args[0]); err != nil {
				return err
			}
			if err := humanf(cmd, "pruned %s\n", args[0]); err != nil {
				return err
			}
			return writeResponse(cmd, "work_pruned", map[string]any{"work": args[0]})
		},
	}
}

func parseDurationFlag(value string) (time.Duration, error) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", value, err)
	}
	return d, nil
}

func workDiscardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "discard <name>",
		Short: "Discard is not implemented in prototype 0",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("discard retention is policy-controlled and not implemented in prototype 0")
		},
	}
}
