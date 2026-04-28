package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Global flags
var (
	namespace string
	server    string
)

var rootCmd = &cobra.Command{
	Use:   "kubecli",
	Short: "kubecli controls the K8-like cluster",
	Long: `kubecli is the kubectl for the K8s-like cluster. It's used to manage resources on the cluster.

		Basic commands:
			get: Display resource(s)
			describe: Show details of a resource
			create: Create resource(s)
			delete: Delete resource(s)
			apply: Apply configuration to resource(s)

		Use "kubecli <command> --help for more information on a command"
		`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "namespace scope")
	rootCmd.PersistentFlags().StringVarP(&server, "server", "s", "http://localhost:5173", "apiserver address")

	rootCmd.AddCommand(getCmd, describeCmd, createCmd, deleteCmd, applyCmd)
}
