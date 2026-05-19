package cmd

import (
	"encoding/json"
	"os"
)

// encodeJSON writes v to stdout with 2-space indentation.
func encodeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
