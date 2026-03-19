package bedrock

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/ellistarn/muse/internal/inference"
)

type stubRuntime struct {
	out *bedrockruntime.ConverseOutput
	err error
}

func (s stubRuntime) Converse(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	return s.out, s.err
}

func TestConverseMessagesPreservesPartialResponseOnTruncation(t *testing.T) {
	client := NewClientWithRuntime(context.Background(), stubRuntime{
		out: &bedrockruntime.ConverseOutput{
			StopReason: types.StopReasonMaxTokens,
			Output: &types.ConverseOutputMemberMessage{
				Value: types.Message{
					Role: types.ConversationRoleAssistant,
					Content: []types.ContentBlock{
						&types.ContentBlockMemberText{Value: "part one "},
						&types.ContentBlockMemberText{Value: "part two"},
					},
				},
			},
			Usage: &types.TokenUsage{
				InputTokens:  aws.Int32(123),
				OutputTokens: aws.Int32(456),
			},
		},
	})

	resp, err := client.ConverseMessages(context.Background(), "system", []inference.Message{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Fatal("expected truncation error")
	}
	if resp == nil {
		t.Fatal("expected partial response")
	}
	if got, want := resp.Text, "part one part two"; got != want {
		t.Fatalf("Text = %q, want %q", got, want)
	}
	if got, want := resp.Usage.InputTokens, 123; got != want {
		t.Fatalf("InputTokens = %d, want %d", got, want)
	}
	if got, want := resp.Usage.OutputTokens, 456; got != want {
		t.Fatalf("OutputTokens = %d, want %d", got, want)
	}
	if !strings.Contains(err.Error(), "response truncated") {
		t.Fatalf("err = %v, want truncation error", err)
	}
}
