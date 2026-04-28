package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <resource> [name]",
	Short: "Display resource(s)",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind := args[0]
		name := ""
		if len(args) == 2 {
			name = args[1]
		}

		url, err := resourceURL(kind, name)
		if err != nil {
			return err
		}
		data, err := doRequest("GET", url, nil)
		if err != nil {
			return err
		}
		return printResource(kind, name == "", data)
	},
}

func printResource(kind string, list bool, data []byte) error {
	path, _, err := resourcePath(kind)
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer tw.Flush()

	switch path {
	case "pods":
		if list {
			var pods []corev1.Pod
			if err := json.Unmarshal(data, &pods); err != nil {
				return err
			}
			fmt.Fprintln(tw, "NAMESPACE\tNAME\tIMAGE\tNODE\tSTATUS")
			for _, p := range pods {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", p.Namespace, p.Name, p.Image, p.NodeName, p.Status)
			}
			return nil
		}
		var p corev1.Pod
		if err := json.Unmarshal(data, &p); err != nil {
			return err
		}
		fmt.Fprintln(tw, "NAMESPACE\tNAME\tIMAGE\tNODE\tSTATUS")
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", p.Namespace, p.Name, p.Image, p.NodeName, p.Status)
	case "nodes":
		if list {
			var nodes []corev1.Node
			if err := json.Unmarshal(data, &nodes); err != nil {
				return err
			}
			fmt.Fprintln(tw, "NAME\tADDRESS\tSTATUS")
			for _, n := range nodes {
				fmt.Fprintf(tw, "%s\t%s\t%s\n", n.Name, n.Address, n.Status)
			}
			return nil
		}
		var n corev1.Node
		if err := json.Unmarshal(data, &n); err != nil {
			return err
		}
		fmt.Fprintln(tw, "NAME\tADDRESS\tSTATUS")
		fmt.Fprintf(tw, "%s\t%s\t%s\n", n.Name, n.Address, n.Status)
	}
	return nil
}
