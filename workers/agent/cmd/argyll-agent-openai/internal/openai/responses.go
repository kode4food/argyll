package openai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type responsesConvo struct {
	client openai.Client
	params responses.ResponseNewParams
}

var (
	ErrResponseFailed = errors.New("OpenAI response failed")
)

var _ worker.Convo = (*responsesConvo)(nil)

// Turn runs one Responses API turn
func (c *responsesConvo) Turn(ctx context.Context) (worker.Turn, error) {
	res, err := c.client.Responses.New(ctx, c.params)
	if err != nil {
		return worker.Turn{}, openAIProblem(err)
	}
	turn := worker.Turn{
		Cost:              reportedCost(res.Usage.JSON.ExtraFields["cost"].Raw()),
		InputTokens:       res.Usage.InputTokens,
		OutputTokens:      res.Usage.OutputTokens,
		CacheInputTokens:  res.Usage.InputTokensDetails.CachedTokens,
		CacheOutputTokens: res.Usage.InputTokensDetails.CacheWriteTokens,
	}
	if res.Status == responses.ResponseStatusFailed {
		return turn, responseProblem(res.Error)
	}
	for _, item := range res.Output {
		input := param.Override[responses.ResponseInputItemUnionParam](
			json.RawMessage(item.RawJSON()),
		)
		c.params.Input.OfInputItemList = append(c.params.Input.OfInputItemList, input)
		if item.Type == "function_call" {
			call := item.AsFunctionCall()
			turn.Calls = append(turn.Calls, worker.ToolCall{
				ID:   call.CallID,
				Name: call.Name,
				Args: json.RawMessage(call.Arguments),
			})
		}
	}
	return turn, nil
}

// Answer adds tool results to the response input
func (c *responsesConvo) Answer(results []worker.ToolResult) {
	for _, r := range results {
		input := responses.ResponseInputItemParamOfFunctionCallOutput(r.Content)
		input.OfFunctionCallOutput.CallID = openai.String(r.CallID)
		c.params.Input.OfInputItemList = append(c.params.Input.OfInputItemList, input)
	}
}

// Nudge adds a user message to the response input
func (c *responsesConvo) Nudge(text string) {
	input := responses.ResponseInputItemParamOfMessage(text, "user")
	c.params.Input.OfInputItemList = append(c.params.Input.OfInputItemList, input)
}

func newResponsesConvo(client openai.Client, ch *worker.Chat) *responsesConvo {
	tools := make([]responses.ToolUnionParam, 0, len(ch.Tools))
	for _, t := range ch.Tools {
		tool := responses.ToolParamOfFunction(t.Name, t.Schema, false)
		tool.OfFunction.Description = openai.String(t.Description)
		tools = append(tools, tool)
	}
	return &responsesConvo{
		client: client,
		params: responses.ResponseNewParams{
			Model:           shared.ResponsesModel(ch.Config.Model),
			MaxOutputTokens: openai.Int(int64(ch.Config.MaxTokens)),
			Store:           openai.Bool(false),
			Include: []responses.ResponseIncludable{
				responses.ResponseIncludableReasoningEncryptedContent,
			},
			Reasoning: shared.ReasoningParam{
				Effort: shared.ReasoningEffort(ch.Config.Effort),
			},
			Input: responses.ResponseNewParamsInputUnion{
				OfInputItemList: responses.ResponseInputParam{
					responses.ResponseInputItemParamOfMessage(ch.System, "developer"),
					responses.ResponseInputItemParamOfMessage(ch.User, "user"),
				},
			},
			Tools: tools,
		},
	}
}

func responseProblem(err responses.ResponseError) *api.ProblemDetails {
	status := http.StatusBadGateway
	switch err.Code {
	case responses.ResponseErrorCodeRateLimitExceeded:
		status = http.StatusTooManyRequests
	case responses.ResponseErrorCodeInvalidPrompt,
		responses.ResponseErrorCodeDataResidencyMismatch,
		responses.ResponseErrorCodeBioPolicy,
		responses.ResponseErrorCodeMisalignmentPolicyViolation:
		status = http.StatusBadRequest
	}
	return &api.ProblemDetails{
		Status: status,
		Title:  ErrResponseFailed.Error(),
		Detail: err.Message,
	}
}
