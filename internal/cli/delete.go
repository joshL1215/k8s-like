package cli

import "github.com/spf13/cobra"

var deleteCmd = &cobra.Command{
	Use:   "delete <resource> <name>",
	Short: "Delete resource(s)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, name := args[0], args[1]

		url, err := resourceURL(kind, name)
		if err != nil {
			return err
		}
		if _, err := doRequest("DELETE", url, nil); err != nil {
			return err
		}
		cmd.Printf("%s %q deleted\n", kind, name)
		return nil
	},
}
