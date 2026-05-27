package cli

import (
	"path/filepath"

	"glyph/internal/store"

	"github.com/spf13/cobra"
)

func vizCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "viz", Short: "Export local Glyph mesh visualizations"}
	cmd.AddCommand(vizExportCmd())
	return cmd
}

func vizExportCmd() *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export a static local visualizer",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			if out == "" {
				defaults, err := store.LoadBootstrapManifest(st.Root)
				if err != nil {
					return err
				}
				out = defaults.Defaults.Viz.Export.Out
				if out == "" {
					out = filepath.Join(st.Dir, "visualizer")
				}
			}
			graph, err := st.WriteVisualizer(out)
			if err != nil {
				return err
			}
			if err := humanf(cmd, "visualizer: %s\n", filepath.Join(out, "index.html")); err != nil {
				return err
			}
			return writeResponse(cmd, "viz_export", map[string]any{
				"out":        out,
				"index":      filepath.Join(out, "index.html"),
				"graph":      filepath.Join(out, "graph.json"),
				"nodes":      len(graph.Nodes),
				"edges":      len(graph.Edges),
				"summary":    graph.Summary,
				"generated":  graph.GeneratedAt,
				"store_root": graph.Root,
			})
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output directory")
	return cmd
}
