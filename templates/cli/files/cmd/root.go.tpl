package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"<% .Module %>/internal/app"
)

var rootCmd = &cobra.Command{
	Use:   "<% .Name %>",
	Short: "<% .Name %> — TODO: one-line description",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return app.Run()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and exit",
	Args:  cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
