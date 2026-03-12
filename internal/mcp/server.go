package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ellistarn/muse/internal/muse"
	"github.com/ellistarn/muse/prompts"
)

// NewServer creates an MCP server with an ask tool.
func NewServer(m *muse.Muse) *server.MCPServer {
	srv := server.NewMCPServer("muse", "0.1.0", server.WithToolCapabilities(false))
	srv.AddTool(
		mcp.NewTool("ask",
			mcp.WithDescription(prompts.Tool),
			mcp.WithString("question", mcp.Required(), mcp.Description("The question to ask")),
		),
		askHandler(m),
	)
	return srv
}

func askHandler(m *muse.Muse) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		question, err := req.RequireString("question")
		if err != nil {
			return nil, err
		}
		answer, err := m.Ask(ctx, question)
		if err != nil {
			return nil, fmt.Errorf("failed to ask: %w", err)
		}
		return mcp.NewToolResultText(answer), nil
	}
}
