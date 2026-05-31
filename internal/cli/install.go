package cli

import (
	"fmt"

	"glyph/internal/install"

	"github.com/spf13/cobra"
)

func installCmd() *cobra.Command {
	var binDir string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install glyph into the user PATH",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := install.Install("", binDir)
			if err != nil {
				return err
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "install", result)
			}
			if result.Installed {
				if err := humanf(cmd, "installed %s\n", result.InstalledPath); err != nil {
					return err
				}
			} else if err := humanf(cmd, "already installed at %s\n", result.InstalledPath); err != nil {
				return err
			}
			switch {
			case result.PathAlreadyConfigured:
				return humanf(cmd, "PATH already includes %s\n", result.BinDir)
			case result.PathConfigured:
				if err := humanf(cmd, "updated PATH configuration for %s\n", result.BinDir); err != nil {
					return err
				}
			default:
				if err := humanf(cmd, "PATH was not changed\n"); err != nil {
					return err
				}
			}
			if len(result.CurrentSessionCommands) > 0 {
				if err := humanf(cmd, "for this terminal session, run:\n"); err != nil {
					return err
				}
				for _, command := range result.CurrentSessionCommands {
					if err := humanf(cmd, "  %s\n", command); err != nil {
						return err
					}
				}
			}
			if len(result.ModifiedFiles) > 0 {
				if err := humanf(cmd, "modified:\n"); err != nil {
					return err
				}
				for _, file := range result.ModifiedFiles {
					if err := humanf(cmd, "  %s\n", file); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&binDir, "bin-dir", "", "directory to install glyph into")
	if err := cmd.RegisterFlagCompletionFunc("bin-dir", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveFilterDirs
	}); err != nil {
		panic(fmt.Sprintf("register bin-dir completion: %v", err))
	}
	return cmd
}
