package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ellistarn/shade/internal/shade"
)

// NewServer creates an MCP server with a single "advise" tool.
func NewServer(s *shade.Shade) *server.MCPServer {
	srv := server.NewMCPServer("shade", "0.1.0", server.WithToolCapabilities(false))
	srv.AddTool(
		mcp.NewTool("shade_advise",
			mcp.WithDescription("Get advice from the shade. The shade responds using its distilled skills and knowledge."),
			mcp.WithString("question", mcp.Required(), mcp.Description("The question to ask")),
		),
		adviseHandler(s),
	)
	return srv
}

func adviseHandler(s *shade.Shade) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		question, err := req.RequireString("question")
		if err != nil {
			return nil, err
		}
		answer, err := s.Advise(ctx, question)
		if err != nil {
			return nil, fmt.Errorf("failed to advise: %w", err)
		}
		return mcp.NewToolResultText(answer), nil
	}
}
