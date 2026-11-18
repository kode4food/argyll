package worker

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/kode4food/argex"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type stepSchemas struct {
	input   string
	outputs []api.Name
	finish  ToolSpec
}

const (
	finishTool   = "finish"
	finishPrompt = "The user message holds your inputs as a JSON object. When you are " +
		"done, call the `" + finishTool + "` tool with the outputs."
	finishNudge = "Call the `" + finishTool + "` tool with the outputs."

	usageInput          = "input_tokens"
	usageOutput         = "output_tokens"
	usageCacheInput     = "cache_input_tokens"
	usageCacheOutput    = "cache_output_tokens"
	usageCost           = "cost"
	usageCostReported   = "cost_reported"
	usageCostCalculated = "cost_calculated"
	usageCostUnpriced   = "cost_unpriced_tokens"
	perMillion          = 1_000_000
)

var (
	ErrTurnLimit = errors.New("no outputs within the turn limit")
)

func (w *Worker) attempt(ctx context.Context, req *api.AgentRequest) api.AgentOutcome {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Millisecond)
	defer cancel()

	// Token counts are reported on every outcome, even one that never reached the model
	usage := api.Usage{
		usageInput:  0,
		usageOutput: 0,
	}
	outputs, err := w.converse(ctx, req, usage)
	if err != nil {
		return api.AgentOutcome{
			Problem: problemOf(err),
			Usage:   usage,
		}
	}
	return api.AgentOutcome{
		Outputs: outputs,
		Usage:   usage,
	}
}

func (w *Worker) converse(
	ctx context.Context, req *api.AgentRequest, usage api.Usage,
) (api.Args, error) {
	cfg := req.Step.Agent
	box, err := openToolbox(ctx, cfg.MCP, w.cfg.Servers)
	if err != nil {
		return nil, err
	}
	defer box.close()

	user, err := json.Marshal(req.Inputs)
	if err != nil {
		return nil, err
	}
	schemas, err := w.stepSchemas(req.Step)
	if err != nil {
		return nil, err
	}
	system := cfg.Prologue + "\n\n" + finishPrompt
	if req.Step.Name != "" {
		system += "\n\nStep: " + string(req.Step.Name)
	}
	if req.Step.Description != "" {
		system += "\n\nStep description: " + req.Step.Description
	}
	system += "\n\nPossible input fields (JSON Schema):\n" + schemas.input
	slices.SortFunc(box.specs, func(a, b ToolSpec) int {
		return strings.Compare(a.Name, b.Name)
	})
	conv, err := w.provider.Open(&Chat{
		Config: cfg,
		System: system,
		User:   string(user),
		Tools:  append(box.specs, schemas.finish),
	})
	if err != nil {
		return nil, err
	}
	for range cfg.MaxTurns {
		t, err := conv.Turn(ctx)
		usage[usageInput] += float64(t.InputTokens)
		usage[usageOutput] += float64(t.OutputTokens)
		if t.CacheInputTokens > 0 {
			usage[usageCacheInput] += float64(t.CacheInputTokens)
		}
		if t.CacheOutputTokens > 0 {
			usage[usageCacheOutput] += float64(t.CacheOutputTokens)
		}
		w.recordCost(usage, &t, cfg)
		if err != nil {
			return nil, err
		}
		if len(t.Calls) == 0 {
			conv.Nudge(finishNudge)
			continue
		}
		results := make([]ToolResult, 0, len(t.Calls))
		for _, call := range t.Calls {
			result, reply, err := handle(ctx, box, call, schemas.outputs)
			if err != nil || result != nil {
				return result, err
			}
			results = append(results, ToolResult{
				CallID:  call.ID,
				Content: reply,
			})
		}
		conv.Answer(results)
	}
	return nil, ErrTurnLimit
}

func (w *Worker) stepSchemas(st *api.Step) (stepSchemas, error) {
	key, err := st.HashKey()
	if err != nil {
		return stepSchemas{}, err
	}
	return w.schemas.Get(key, func() (stepSchemas, error) {
		input, err := inputSchema(st)
		if err != nil {
			return stepSchemas{}, err
		}
		outputs := outputNames(st)
		return stepSchemas{
			input:   string(input),
			outputs: outputs,
			finish:  finishSpec(outputs),
		}, nil
	})
}

func (w *Worker) recordCost(usage api.Usage, t *Turn, cfg *api.AgentConfig) {
	if t.Cost != nil {
		addCharge(usage, usageCostReported, *t.Cost)
		return
	}
	price := w.cfg.Prices[cfg.Provider+"/"+cfg.Model]
	input := max(t.InputTokens-t.CacheInputTokens-t.CacheOutputTokens, 0)
	for _, charge := range []struct {
		tokens int64
		price  *float64
	}{
		{tokens: input, price: price.Input},
		{tokens: t.OutputTokens, price: price.Output},
		{tokens: t.CacheInputTokens, price: price.CacheInput},
		{tokens: t.CacheOutputTokens, price: price.CacheOutput},
	} {
		if charge.tokens == 0 {
			continue
		}
		if charge.price == nil {
			usage[usageCostUnpriced] += float64(charge.tokens)
			continue
		}
		addCharge(usage, usageCostCalculated, tokenCost(float64(charge.tokens), *charge.price))
	}
}

func inputSchema(st *api.Step) ([]byte, error) {
	properties := map[api.Name]struct{}{}
	required := []api.Name{}
	for name, attr := range st.Attributes {
		if !attr.IsRuntimeInput() {
			continue
		}
		if attr.Collect() == api.InputCollectNone && attr.OptionalDefault() == "" {
			continue
		}
		mapped, _ := st.MappedName(name)
		properties[mapped] = struct{}{}
		if attr.IsRequired() || attr.IsConst() || attr.IsFeed() {
			required = append(required, mapped)
		}
	}
	slices.Sort(required)
	return json.Marshal(map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	})
}

func outputNames(st *api.Step) []api.Name {
	var res []api.Name
	for _, name := range st.GetOutputArgs() {
		mapped, _ := st.MappedName(name)
		res = append(res, mapped)
	}
	slices.Sort(res)
	return res
}

// handle answers one tool call. A finish call naming exactly the outputs ends the attempt, and any
// other mistake is a reply the model can correct
func handle(
	ctx context.Context, box *toolbox, call ToolCall, outputs []api.Name,
) (api.Args, string, error) {
	var args api.Args
	if err := json.Unmarshal(call.Args, &args); err != nil || args == nil {
		return nil, "Error: arguments must be a JSON object", nil
	}
	switch {
	case call.Name == finishTool && !sameNames(args, outputs):
		return nil, "Error: finish takes exactly " + joinNames(outputs), nil
	case call.Name == finishTool:
		return args, "", nil
	case !box.has(call.Name):
		return nil, "Error: unknown tool " + call.Name, nil
	}
	reply, err := box.call(ctx, call.Name, args)
	return nil, reply, err
}

func finishSpec(outputs []api.Name) ToolSpec {
	props := make(map[string]any, len(outputs))
	required := make([]any, 0, len(outputs))
	for _, n := range outputs {
		props[string(n)] = map[string]any{}
		required = append(required, string(n))
	}
	return ToolSpec{
		Name:        finishTool,
		Description: "Report the outputs and end the task",
		Schema: map[string]any{
			"type":                 "object",
			"properties":           props,
			"required":             required,
			"additionalProperties": false,
		},
	}
}

// problemOf decides how the engine treats a failed attempt. A provider's own status stands, the
// turn limit and a missing key fail permanently, and anything else, including the timeout, retries
func problemOf(err error) *api.ProblemDetails {
	status := http.StatusBadGateway
	var problem *api.ProblemDetails
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		status = http.StatusGatewayTimeout
	case errors.Is(err, ErrTurnLimit), errors.Is(err, ErrUnknownServer),
		errors.Is(err, ErrDuplicateTool):
		status = http.StatusUnprocessableEntity
	case errors.Is(err, ErrMissingAPIKey):
		status = http.StatusUnauthorized
	case errors.As(err, &problem):
		return problem
	}
	return &api.ProblemDetails{
		Status: status,
		Detail: err.Error(),
	}
}

func addCharge(usage api.Usage, source string, cost float64) {
	for _, name := range []string{usageCost, source} {
		if sum, ok := argex.Add(usage[name], cost); ok {
			usage[name] = sum
			continue
		}
		usage[name] += cost
	}
}

// tokenCost prices tokens with decimal arithmetic to avoid binary rounding artifacts
func tokenCost(tokens, perMillionPrice float64) float64 {
	product, ok := argex.Mul(tokens, perMillionPrice)
	if !ok {
		product = tokens * perMillionPrice
	}
	if res, ok := argex.Div(product, perMillion); ok {
		return res
	}
	return product / perMillion
}

func sameNames(args api.Args, outputs []api.Name) bool {
	if len(args) != len(outputs) {
		return false
	}
	for _, n := range outputs {
		if _, ok := args[n]; !ok {
			return false
		}
	}
	return true
}

func joinNames(names []api.Name) string {
	strs := make([]string, 0, len(names))
	for _, n := range names {
		strs = append(strs, string(n))
	}
	return strings.Join(strs, ", ")
}
