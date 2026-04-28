package cli

import (
	"fmt"
	"os"
	"strings"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
	"github.com/spf13/cobra"
)

var applyFile string

var applyCmd = &cobra.Command{
	Use:   "apply -f <file>",
	Short: "Apply configuration to resource(s)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if applyFile == "" {
			return fmt.Errorf("a manifest file must be provided with -f")
		}
		data, err := os.ReadFile(applyFile)
		if err != nil {
			return err
		}
		kind, obj, err := decodeManifest(data)
		if err != nil {
			return err
		}
		name, err := objectName(obj)
		if err != nil {
			return err
		}

		url, err := resourceURL(kind, name)
		if err != nil {
			return err
		}
		if _, err := doRequest("PUT", url, obj); err != nil {
			if !strings.Contains(err.Error(), "not") {
				return err
			}
			createURL, err := resourceURL(kind, "")
			if err != nil {
				return err
			}
			if _, err := doRequest("POST", createURL, obj); err != nil {
				return err
			}
			cmd.Printf("%s %q created\n", kind, name)
			return nil
		}
		cmd.Printf("%s %q configured\n", kind, name)
		return nil
	},
}

func objectName(obj any) (string, error) {
	switch o := obj.(type) {
	case *corev1.Pod:
		return o.Name, nil
	case *corev1.Node:
		return o.Name, nil
	}
	return "", fmt.Errorf("unsupported object type")
}

func init() {
	applyCmd.Flags().StringVarP(&applyFile, "filename", "f", "", "manifest file (JSON)")
}
