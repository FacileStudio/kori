package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

func newSandboxCmd() *cobra.Command { //nolint:unusedfunc
	return &cobra.Command{
		Use:   "sandbox <vm-name>",
		Short: "Start kori inside a boite VM sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("sandbox integration not implemented yet")
		},
	}
}
