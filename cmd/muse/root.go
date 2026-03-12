package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var bucket string

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "muse",
		Short:         "The distilled essence of how you think",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.PersistentFlags().StringVar(&bucket, "bucket", os.Getenv("MUSE_BUCKET"), "S3 bucket name (or set MUSE_BUCKET)")
	cmd.AddCommand(newPushCmd())
	cmd.AddCommand(newDreamCmd())
	cmd.AddCommand(newInspectCmd())
	cmd.AddCommand(newListenCmd())
	cmd.AddCommand(newAskCmd())
	return cmd
}

func requireBucket() error {
	if bucket == "" {
		return fmt.Errorf("bucket is required: use --bucket or set MUSE_BUCKET")
	}
	return nil
}
