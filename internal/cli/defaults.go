package cli

import (
	"glyph/internal/store"

	"github.com/spf13/cobra"
)

func commandDefaults() (*store.BootstrapManifest, error) {
	st, err := openStore()
	if err != nil {
		return nil, err
	}
	defer st.Close()
	return store.LoadBootstrapManifest(st.Root)
}

func flagChanged(cmd *cobra.Command, name string) bool {
	flag := cmd.Flags().Lookup(name)
	return flag != nil && flag.Changed
}
