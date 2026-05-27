package main

import (
	"errors"
	"fmt"
	"os"

	"glyph/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		var reported interface{ JSONReported() bool }
		if !errors.As(err, &reported) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
