package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show gokart configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newConfigShowCommand())
	return cmd
}

func newConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print where gokart stores data",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cfgDir, err := os.UserConfigDir()
			if err != nil {
				cfgDir = "(unavailable: " + err.Error() + ")"
			}
			fmt.Printf("Version:     %s\n", gokartVersion)
			fmt.Printf("Config dir:  %s\n", cfgDir)
			fmt.Printf("Binary:      %s\n", os.Args[0])
		},
	}
}
