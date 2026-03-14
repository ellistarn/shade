package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ellistarn/muse/internal/storage"
)

var validCategories = map[string]bool{
	"memories":    true,
	"reflections": true,
	"souls":       true,
}

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync <src> <dst> [category...]",
		Short: "Sync data between storage backends",
		Long: `Copies data from one storage backend to another. Additive — items in the
destination that don't exist in the source are left alone.

Endpoints:
  s3      S3 bucket (requires MUSE_BUCKET)
  local   Local filesystem (~/.muse/)

Categories:
  memories      Raw conversation sessions
  reflections   Per-session observation summaries
  souls         Timestamped soul versions

If no categories are specified, all data is synced.`,
		Example: `  muse sync s3 local              # pull everything from S3
  muse sync local s3              # push everything to S3
  muse sync s3 local memories     # pull only raw sessions
  muse sync s3 local souls        # pull only soul versions`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			srcName, dstName := args[0], args[1]
			categories := args[2:]

			for _, c := range categories {
				if !validCategories[c] {
					return fmt.Errorf("unknown category %q (valid: memories, reflections, souls)", c)
				}
			}

			src, err := resolveEndpoint(ctx, srcName)
			if err != nil {
				return fmt.Errorf("source %q: %w", srcName, err)
			}
			dst, err := resolveEndpoint(ctx, dstName)
			if err != nil {
				return fmt.Errorf("destination %q: %w", dstName, err)
			}

			return storage.Sync(ctx, src, dst, categories, os.Stdout)
		},
	}
	return cmd
}

func resolveEndpoint(ctx context.Context, name string) (storage.Store, error) {
	switch name {
	case "s3":
		if bucket == "" {
			return nil, fmt.Errorf("MUSE_BUCKET must be set to use s3 endpoint")
		}
		return storage.NewS3Store(ctx, bucket)
	case "local":
		return storage.NewLocalStore()
	default:
		return nil, fmt.Errorf("unknown endpoint %q (valid: s3, local)", name)
	}
}
