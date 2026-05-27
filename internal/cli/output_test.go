package cli

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

func TestWriteJSONErrorUsesStableCode(t *testing.T) {
	cmd := &cobra.Command{Use: "glyph"}
	cmd.PersistentFlags().Bool("json", false, "")
	if err := cmd.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}

	errOut := new(bytes.Buffer)
	cmd.SetErr(errOut)

	if err := writeJSONError(cmd, sql.ErrNoRows); err != nil {
		t.Fatal(err)
	}

	var got errorResponse
	if err := json.Unmarshal(errOut.Bytes(), &got); err != nil {
		t.Fatal(err)
	}

	if got.OK {
		t.Fatalf("got OK=true, want false")
	}
	if got.Error.Code != "not_found" {
		t.Fatalf("got code %q, want not_found", got.Error.Code)
	}
}
