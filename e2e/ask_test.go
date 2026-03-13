package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/ellistarn/muse/internal/bedrock"
	"github.com/ellistarn/muse/internal/muse"
)

// mockRuntime implements bedrock.Runtime with canned responses.
type mockRuntime struct {
	calls     []converseCall
	responses []bedrockruntime.ConverseOutput
	callIndex int
}

type converseCall struct {
	system   string
	messages int
}

func (m *mockRuntime) Converse(_ context.Context, params *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	var system string
	for _, block := range params.System {
		if tb, ok := block.(*brtypes.SystemContentBlockMemberText); ok {
			system = tb.Value
		}
	}
	m.calls = append(m.calls, converseCall{system: system, messages: len(params.Messages)})
	if m.callIndex >= len(m.responses) {
		return nil, nil
	}
	resp := m.responses[m.callIndex]
	m.callIndex++
	return &resp, nil
}

func textResponse(text string) bedrockruntime.ConverseOutput {
	return bedrockruntime.ConverseOutput{
		StopReason: brtypes.StopReasonEndTurn,
		Output: &brtypes.ConverseOutputMemberMessage{
			Value: brtypes.Message{
				Role: brtypes.ConversationRoleAssistant,
				Content: []brtypes.ContentBlock{
					&brtypes.ContentBlockMemberText{Value: text},
				},
			},
		},
		Usage: &brtypes.TokenUsage{
			InputTokens:  aws.Int32(100),
			OutputTokens: aws.Int32(50),
		},
	}
}

func TestAskWithSoul(t *testing.T) {
	runtime := &mockRuntime{responses: []bedrockruntime.ConverseOutput{
		textResponse("Use kebab-case for your file names."),
	}}

	ctx := context.Background()
	bedrockClient := bedrock.NewClientWithRuntime(ctx, runtime)
	m := muse.NewForTest(bedrockClient, "Use kebab-case for file names. Always wrap errors with context.")

	answer, err := m.Ask(ctx, "how should I name files?")
	if err != nil {
		t.Fatalf("Ask() error: %v", err)
	}
	if answer != "Use kebab-case for your file names." {
		t.Errorf("answer = %q, want %q", answer, "Use kebab-case for your file names.")
	}

	// Single Bedrock call — no tool use loop
	if len(runtime.calls) != 1 {
		t.Fatalf("Bedrock calls = %d, want 1", len(runtime.calls))
	}
	// Soul content should be in system prompt
	if !strings.Contains(runtime.calls[0].system, "kebab-case") {
		t.Error("system prompt missing soul content")
	}
	if !strings.Contains(runtime.calls[0].system, "wrap errors") {
		t.Error("system prompt missing soul content about error handling")
	}
	// Single user message
	if runtime.calls[0].messages != 1 {
		t.Errorf("messages = %d, want 1", runtime.calls[0].messages)
	}
}

func TestAskEmptySoul(t *testing.T) {
	runtime := &mockRuntime{responses: []bedrockruntime.ConverseOutput{
		textResponse("I don't have any knowledge to draw on for that."),
	}}

	ctx := context.Background()
	bedrockClient := bedrock.NewClientWithRuntime(ctx, runtime)
	m := muse.NewForTest(bedrockClient, "")

	answer, err := m.Ask(ctx, "how do you handle errors?")
	if err != nil {
		t.Fatalf("Ask() error: %v", err)
	}
	if answer != "I don't have any knowledge to draw on for that." {
		t.Errorf("answer = %q", answer)
	}

	if len(runtime.calls) != 1 {
		t.Fatalf("Bedrock calls = %d, want 1", len(runtime.calls))
	}
	if !strings.Contains(runtime.calls[0].system, "No soul document available") {
		t.Error("system prompt should indicate no soul available")
	}
}
