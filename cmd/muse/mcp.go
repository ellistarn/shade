package main

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	mcpserver "github.com/ellistarn/muse/internal/mcp"
	"github.com/ellistarn/muse/internal/muse"
)

func newListenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "listen",
		Short: "Start the muse MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireBucket(); err != nil {
				return err
			}
			ctx := cmd.Context()
			m, err := muse.New(ctx, bucket)
			if err != nil {
				return err
			}
			srv := mcpserver.NewServer(m)
			return server.ServeStdio(srv)
		},
	}
}
