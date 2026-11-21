package bedrock

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type convo struct {
	client *bedrockruntime.Client
	input  bedrockruntime.ConverseInput
}

var (
	ErrNoMessage = errors.New("model returned no message")
)

var _ worker.Convo = (*convo)(nil)

// Turn runs one Bedrock model turn
func (c *convo) Turn(ctx context.Context) (worker.Turn, error) {
	out, err := c.client.Converse(ctx, &c.input)
	if err != nil {
		return worker.Turn{}, problem(err)
	}
	res := usage(out.Usage)
	switch out.StopReason {
	case types.StopReasonMaxTokens, types.StopReasonGuardrailIntervened,
		types.StopReasonContentFiltered, types.StopReasonModelContextWindowExceeded,
		types.StopReasonMalformedModelOutput, types.StopReasonMalformedToolUse:
		return res, fmt.Errorf("%w: bedrock: %s", worker.ErrModelStopped, out.StopReason)
	}
	msg, ok := out.Output.(*types.ConverseOutputMemberMessage)
	if !ok {
		return res, ErrNoMessage
	}
	c.input.Messages = append(c.input.Messages, msg.Value)
	for _, block := range msg.Value.Content {
		use, ok := block.(*types.ContentBlockMemberToolUse)
		if !ok {
			continue
		}
		args, err := use.Value.Input.MarshalSmithyDocument()
		if err != nil {
			return res, err
		}
		res.Calls = append(res.Calls, worker.ToolCall{
			ID:   aws.ToString(use.Value.ToolUseId),
			Name: aws.ToString(use.Value.Name),
			Args: args,
		})
	}
	return res, nil
}

// Answer adds tool replies to the conversation
func (c *convo) Answer(results []worker.ToolResult) {
	blocks := make([]types.ContentBlock, 0, len(results))
	for _, r := range results {
		blocks = append(blocks, &types.ContentBlockMemberToolResult{
			Value: types.ToolResultBlock{
				ToolUseId: aws.String(r.CallID),
				Content: []types.ToolResultContentBlock{
					&types.ToolResultContentBlockMemberText{Value: r.Content},
				},
			},
		})
	}
	c.input.Messages = append(c.input.Messages, types.Message{
		Role:    types.ConversationRoleUser,
		Content: blocks,
	})
}

// Nudge adds an instruction to the conversation
func (c *convo) Nudge(text string) {
	c.input.Messages = append(c.input.Messages, userText(text))
}

// usage counts cached tokens as input, since Bedrock reports them apart from the rest
func usage(u *types.TokenUsage) worker.Turn {
	if u == nil {
		return worker.Turn{}
	}
	read := int64(aws.ToInt32(u.CacheReadInputTokens))
	write := int64(aws.ToInt32(u.CacheWriteInputTokens))
	return worker.Turn{
		InputTokens:       int64(aws.ToInt32(u.InputTokens)) + read + write,
		OutputTokens:      int64(aws.ToInt32(u.OutputTokens)),
		CacheInputTokens:  read,
		CacheOutputTokens: write,
	}
}

func userText(text string) types.Message {
	return types.Message{
		Role:    types.ConversationRoleUser,
		Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: text}},
	}
}
