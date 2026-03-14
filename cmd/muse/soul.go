package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ellistarn/muse/internal/bedrock"
	"github.com/ellistarn/muse/internal/inference"
	"github.com/ellistarn/muse/internal/storage"
)

func newSoulCmd() *cobra.Command {
	var diff bool
	cmd := &cobra.Command{
		Use:   "soul",
		Short: "Print soul.md",
		Long: `Prints your current soul document to stdout. If no soul exists yet, prompts
you to run 'muse dream'.

Use --diff to summarize what changed since the last dream.`,
		Example: `  muse soul          # print the soul
  muse soul --diff   # summarize what changed since the last dream`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			store, err := newStore(ctx)
			if err != nil {
				return err
			}

			if diff {
				return runDiff(cmd, store)
			}

			soul, err := store.GetSoul(ctx)
			if err != nil {
				if !storage.IsNotFound(err) {
					return fmt.Errorf("failed to load soul: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "No soul found. Run 'muse dream' to generate one from memories.")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(soul))
			fmt.Fprintf(cmd.ErrOrStderr(), "soul.md: ~%d tokens\n", inference.EstimateTokens(soul))
			return nil
		},
	}
	cmd.Flags().BoolVar(&diff, "diff", false, "summarize what changed since the last dream")
	return cmd
}

func runDiff(cmd *cobra.Command, store storage.Store) error {
	ctx := cmd.Context()

	souls, err := store.ListSouls(ctx)
	if err != nil {
		return fmt.Errorf("failed to list soul history: %w", err)
	}
	if len(souls) < 2 {
		return fmt.Errorf("need at least 2 soul versions to diff; only found %d", len(souls))
	}
	prevTimestamp := souls[len(souls)-2]

	prev, err := store.GetSoulVersion(ctx, prevTimestamp)
	if err != nil {
		return fmt.Errorf("failed to load soul version %s: %w", prevTimestamp, err)
	}
	current, err := store.GetSoulVersion(ctx, souls[len(souls)-1])
	if err != nil {
		if !storage.IsNotFound(err) {
			return fmt.Errorf("failed to load current soul: %w", err)
		}
		current = ""
	}

	if prev == "" && current == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "No soul in either snapshot.")
		return nil
	}

	llm, err := bedrock.NewClient(ctx, bedrock.ModelSonnet)
	if err != nil {
		return err
	}

	prompt := `Compare the previous and current soul documents. Summarize what changed in a few concise bullet points: which sections were added, removed, or meaningfully revised. Focus on substance, not formatting.`

	summary, usage, err := llm.Converse(ctx, prompt, "Previous:\n\n"+prev+"\n\n---\n\nCurrent:\n\n"+current)
	if err != nil {
		return fmt.Errorf("failed to generate diff summary: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Changes since %s:\n\n%s\n", prevTimestamp, strings.TrimSpace(summary))
	fmt.Fprintf(cmd.ErrOrStderr(), "tokens: %d in / %d out · $%.4f\n",
		usage.InputTokens, usage.OutputTokens, usage.Cost())
	return nil
}
