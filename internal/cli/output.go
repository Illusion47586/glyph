package cli

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type outputMode struct {
	JSON bool
}

type response struct {
	OK   bool   `json:"ok"`
	Type string `json:"type,omitempty"`
	Data any    `json:"data,omitempty"`
}

type errorResponse struct {
	OK    bool          `json:"ok"`
	Error responseError `json:"error"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeResponse(cmd *cobra.Command, typ string, data any) error {
	mode := modeFrom(cmd)
	if !mode.JSON {
		return nil
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(response{OK: true, Type: typ, Data: data})
}

func writeJSONError(cmd *cobra.Command, err error) error {
	enc := json.NewEncoder(cmd.ErrOrStderr())
	enc.SetIndent("", "  ")
	return enc.Encode(errorResponse{
		OK: false,
		Error: responseError{
			Code:    errorCode(err),
			Message: err.Error(),
		},
	})
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "not_found"
	case strings.Contains(err.Error(), "required"):
		return "invalid_argument"
	case strings.Contains(err.Error(), "not implemented"):
		return "not_implemented"
	default:
		return "error"
	}
}

func modeFrom(cmd *cobra.Command) outputMode {
	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")
	return outputMode{JSON: jsonOut}
}

func humanf(cmd *cobra.Command, format string, args ...any) error {
	if modeFrom(cmd).JSON {
		return nil
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), format, args...)
	return err
}
