package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe <resource> <name>",
	Short: "Show details of a resource",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, name := args[0], args[1]

		url, err := resourceURL(kind, name)
		if err != nil {
			return err
		}
		data, err := doRequest("GET", url, nil)
		if err != nil {
			return err
		}

		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		out, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		cmd.Println(string(out))
		return nil
	},
}
