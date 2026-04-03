package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDiscoverCommand() *cobra.Command {
	var source string
	var platform string

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover VM inventory from a hypervisor",
		Long:  "Discover VM inventory from a source hypervisor and normalize it into the Viaduct schema.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "Discovery not yet implemented for %s at %s\n", platform, source)
			return nil
		},
	}

	cmd.Flags().StringVarP(&source, "source", "s", "", "Source hypervisor endpoint")
	cmd.Flags().StringVarP(&platform, "type", "t", "", "Source platform type")

	cobra.CheckErr(cmd.MarkFlagRequired("source"))
	cobra.CheckErr(cmd.MarkFlagRequired("type"))

	return cmd
}
