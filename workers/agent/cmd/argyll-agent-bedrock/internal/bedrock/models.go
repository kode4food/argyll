package bedrock

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

// Models lists the region's on-demand text models and active inference profiles, leaving out
// retired models
func (p *Provider) Models(ctx context.Context, _ api.ProviderID) ([]api.AgentModel, error) {
	found, err := p.control.ListFoundationModels(ctx, &bedrock.ListFoundationModelsInput{
		ByOutputModality: types.ModelModalityText,
	})
	if err != nil {
		return nil, problem(err)
	}
	now := time.Now()
	excluded := util.Set[string]{}
	profiles := map[string]string{}
	var models []api.AgentModel
	for _, m := range found.ModelSummaries {
		id := aws.ToString(m.ModelId)
		textOnly := len(m.OutputModalities) == 1 && m.OutputModalities[0] == types.ModelModalityText
		if retired(m.ModelLifecycle, now) || !textOnly {
			excluded.Add(id)
			continue
		}
		onDemand := slices.Contains(m.InferenceTypesSupported, types.InferenceTypeOnDemand)
		if !onDemand {
			continue
		}
		if caps, ok := p.capabilities(id); ok {
			models = append(models, api.AgentModel{
				ID:      api.ModelID(id),
				Efforts: caps.Efforts,
			})
		}
	}
	baseCount := len(models)
	pages := bedrock.NewListInferenceProfilesPaginator(
		p.control, &bedrock.ListInferenceProfilesInput{},
	)
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)
		if err != nil {
			return models[:baseCount], problem(err)
		}
		for _, ip := range page.InferenceProfileSummaries {
			if ip.Status != types.InferenceProfileStatusActive {
				continue
			}
			id, ok := profileModel(ip.Models)
			caps, supported := p.capabilities(id)
			if !ok || !supported || excluded.Contains(id) {
				continue
			}
			models = append(models, api.AgentModel{
				ID:      api.ModelID(aws.ToString(ip.InferenceProfileId)),
				Efforts: caps.Efforts,
			})
			profiles[aws.ToString(ip.InferenceProfileId)] = id
			if arn := aws.ToString(ip.InferenceProfileArn); arn != "" {
				profiles[arn] = id
			}
		}
	}
	p.profiles.Store(&profiles)
	return models, nil
}

func (p *Provider) capabilities(id string) (*worker.Capabilities, bool) {
	caps := p.models[worker.ModelKey{
		Provider: Name,
		Model:    api.ModelID(id),
	}]
	return &caps, !caps.Hidden
}

// requestFields maps effort to each model family's Converse fields using local capabilities
func (p *Provider) requestFields(id string, effort api.Effort) (document.Interface, error) {
	base, caps, err := p.resolveModel(id)
	if err != nil {
		return nil, err
	}
	if effort == "" {
		return nil, nil
	}
	if !slices.Contains(caps.Efforts, effort) {
		return nil, fmt.Errorf("%w: %s: %s", ErrInvalidEffort, id, effort)
	}
	fields := map[string]any{}
	switch {
	case strings.Contains(base, "anthropic.claude-"):
		fields["output_config"] = map[string]any{"effort": effort}
	case strings.Contains(base, "amazon.nova-"):
		fields["reasoningConfig"] = map[string]any{
			"type":               "enabled",
			"maxReasoningEffort": effort,
		}
	case strings.Contains(base, "openai."):
		fields["reasoning_effort"] = effort
	default:
		return nil, fmt.Errorf("%w: %s: %s", ErrInvalidEffort, id, effort)
	}
	return document.NewLazyDocument(fields), nil
}

// resolveModel uses local capabilities and the last complete profile snapshot. An ID found in
// neither is treated as a foundation model with default capabilities
func (p *Provider) resolveModel(id string) (string, *worker.Capabilities, error) {
	caps, ok := p.models[worker.ModelKey{
		Provider: Name,
		Model:    api.ModelID(id),
	}]
	if ok && caps.Hidden {
		return "", nil, fmt.Errorf("%w: %s", ErrUnsupportedModel, id)
	}
	if ok {
		return id, &caps, nil
	}
	profiles := p.profiles.Load()
	if profiles == nil {
		return "", nil, fmt.Errorf("%w: %s", ErrProfilesUnavailable, id)
	}
	base, ok := (*profiles)[id]
	if !ok {
		return id, &worker.Capabilities{}, nil
	}
	caps = p.models[worker.ModelKey{
		Provider: Name,
		Model:    api.ModelID(base),
	}]
	if caps.Hidden {
		return "", nil, fmt.Errorf("%w: %s", ErrUnsupportedModel, id)
	}
	return base, &caps, nil
}

func retired(life *types.FoundationModelLifecycle, now time.Time) bool {
	return life != nil && life.EndOfLifeTime != nil && !now.Before(*life.EndOfLifeTime)
}

// profileModel accepts only profiles whose regional targets share one foundation model
func profileModel(models []types.InferenceProfileModel) (string, bool) {
	var id string
	for _, m := range models {
		_, next, ok := strings.Cut(aws.ToString(m.ModelArn), ":foundation-model/")
		if !ok || next == "" || id != "" && id != next {
			return "", false
		}
		id = next
	}
	return id, id != ""
}
