//go:build mac
// +build mac

package main

import (
	"github.com/AudiusProject/audius-d/pkg/statusbar"
	"github.com/spf13/cobra"
)

var sbCmd = &cobra.Command{
	Use:   "statusbar",
	Short: "Run mac status bar",
	RunE: func(cmd *cobra.Command, args []string) error {
		statusbar.RunStatusBar()
		return nil
	},
}
