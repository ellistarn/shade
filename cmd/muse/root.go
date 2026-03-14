package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	muselog "github.com/ellistarn/muse/internal/log"
	"github.com/ellistarn/muse/internal/storage"
)

var (
	bucket string
	debug  bool
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "muse",
		Short: "The distilled essence of how you think",
		Long: `A muse absorbs your memories from agent interactions, distills them into a
soul document, and embodies your unique thought processes when asked questions.

Workflow:

  1. muse dream      Discover memories, reflect, and distill a soul document
  2. muse soul       Print the soul document
  3. muse ask        Ask your muse a question (stateless, one-shot)
  4. muse listen     Start an MCP server so agents can ask your muse

Getting started:

  muse dream && muse soul

Data is stored locally at ~/.muse/ by default. Set MUSE_BUCKET to use S3 instead.

Run "muse listen --help" for MCP server configuration.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			muselog.Init(debug)
		},
	}
	cmd.PersistentFlags().StringVar(&bucket, "bucket", os.Getenv("MUSE_BUCKET"), "S3 bucket name (or set MUSE_BUCKET)")
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	cmd.AddCommand(newDreamCmd())
	cmd.AddCommand(newSoulCmd())
	cmd.AddCommand(newListenCmd())
	cmd.AddCommand(newAskCmd())
	cmd.AddCommand(newSyncCmd())
	return cmd
}

// newStore returns an S3-backed store when a bucket is configured,
// otherwise a local filesystem store rooted at ~/.muse/.
func newStore(ctx context.Context) (storage.Store, error) {
	if bucket != "" {
		slog.Debug("using S3 storage", "bucket", bucket)
		return storage.NewS3Store(ctx, bucket)
	}
	store, err := storage.NewLocalStore()
	if err != nil {
		return nil, err
	}
	slog.Debug("using local storage", "path", store.Root())
	return store, nil
}
