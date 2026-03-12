package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ellistarn/shade/internal/bedrock"
	"github.com/ellistarn/shade/internal/dream"
	"github.com/ellistarn/shade/internal/log"
	"github.com/ellistarn/shade/internal/storage"
)

func newRelearnCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "relearn",
		Short: "Re-synthesize skills from persisted reflections",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireBucket(); err != nil {
				return err
			}
			ctx := cmd.Context()
			store, err := storage.NewClient(ctx, bucket)
			if err != nil {
				return err
			}
			llm, err := bedrock.NewClient(ctx, bedrock.ModelSonnet)
			if err != nil {
				return err
			}
			log.Printf("Using model %s\n", llm.Model())
			result, err := dream.Relearn(ctx, store, llm)
			if err != nil {
				return err
			}
			for _, w := range result.Warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Produced %d skills (%dk input, %dk output tokens, $%.2f)\n",
				result.Skills, result.Usage.InputTokens/1000, result.Usage.OutputTokens/1000, result.Usage.Cost())
			return nil
		},
	}
}
