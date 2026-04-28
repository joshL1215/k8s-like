package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var createFile string

var createCmd = &cobra.Command{
	Use:   "create -f <file>",
	Short: "Create resource(s)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if createFile == "" {
			return fmt.Errorf("a manifest file must be provided with -f")
		}
		data, err := os.ReadFile(createFile)
		if err != nil {
			return err
		}
		kind, obj, err := decodeManifest(data)
		if err != nil {
			return err
		}
		url, err := resourceURL(kind, "")
		if err != nil {
			return err
		}
		if _, err := doRequest("POST", url, obj); err != nil {
			return err
		}
		cmd.Printf("%s created\n", kind)
		return nil
	},
}

func init() {
	createCmd.Flags().StringVarP(&createFile, "filename", "f", "", "manifest file (JSON)")
}
