package bedrock

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

// Provider runs agent work through the model's verified Amazon Bedrock API
type Provider struct {
	region   string
	runtime  *bedrockruntime.Client
	control  *bedrock.Client
	models   map[worker.ModelKey]worker.Capabilities
	profiles atomic.Pointer[map[string]string]
}

// Name identifies the Amazon Bedrock provider
const Name api.ProviderID = "bedrock"

var (
	ErrUnsupportedModel    = errors.New("model does not support agent tool use")
	ErrInvalidEffort       = errors.New("unsupported model effort")
	ErrProfilesUnavailable = errors.New("inference profiles have not been discovered")
)

var _ worker.Provider = (*Provider)(nil)

// NewProvider returns a Bedrock provider using the standard AWS configuration
func NewProvider(cfg *worker.Config) (*Provider, error) {
	awsCfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}
	return &Provider{
		region:  awsCfg.Region,
		runtime: bedrockruntime.NewFromConfig(awsCfg),
		control: bedrock.NewFromConfig(awsCfg),
		models:  cfg.Capabilities,
	}, nil
}

// Supports reports whether this adapter serves the named provider
func (*Provider) Supports(name api.ProviderID) bool {
	return name == Name
}

// Names lists Bedrock when an AWS region is configured
func (p *Provider) Names() []api.ProviderID {
	if p.region != "" {
		return []api.ProviderID{Name}
	}
	return nil
}

// Open starts a Bedrock conversation with the task's context and tools
func (p *Provider) Open(ch *worker.Chat) (worker.Convo, error) {
	if p.region == "" {
		return nil, fmt.Errorf("%w: %s", worker.ErrMissingAPIKey, "AWS_REGION")
	}
	_, caps, err := p.resolveModel(string(ch.Config.Model))
	if err != nil {
		return nil, problem(err)
	}
	if ch.Config.Effort != "" && !slices.Contains(caps.Efforts, ch.Config.Effort) {
		err := fmt.Errorf("%w: %s: %s", ErrInvalidEffort, ch.Config.Model, ch.Config.Effort)
		return nil, problem(err)
	}
	return p.openConverse(ch)
}

func (p *Provider) openConverse(ch *worker.Chat) (worker.Convo, error) {
	fields, err := p.requestFields(string(ch.Config.Model), ch.Config.Effort)
	if err != nil {
		return nil, problem(err)
	}
	tools := make([]types.Tool, 0, len(ch.Tools))
	for _, t := range ch.Tools {
		tools = append(tools, &types.ToolMemberToolSpec{
			Value: types.ToolSpecification{
				Name:        aws.String(t.Name),
				Description: aws.String(t.Description),
				InputSchema: &types.ToolInputSchemaMemberJson{
					Value: document.NewLazyDocument(t.Schema),
				},
			},
		})
	}
	return &convo{
		client: p.runtime,
		input: bedrockruntime.ConverseInput{
			ModelId: aws.String(string(ch.Config.Model)),
			System: []types.SystemContentBlock{
				&types.SystemContentBlockMemberText{Value: ch.System},
			},
			Messages: []types.Message{userText(ch.User)},
			InferenceConfig: &types.InferenceConfiguration{
				MaxTokens: aws.Int32(int32(ch.Config.MaxTokens)),
			},
			ToolConfig:                   &types.ToolConfiguration{Tools: tools},
			AdditionalModelRequestFields: fields,
		},
	}, nil
}

func problem(err error) error {
	if errors.Is(err, ErrProfilesUnavailable) {
		return errors.Join(err, &api.ProblemDetails{
			Status: http.StatusServiceUnavailable,
			Detail: err.Error(),
		})
	}
	if errors.Is(err, ErrUnsupportedModel) || errors.Is(err, ErrInvalidEffort) {
		return errors.Join(err, &api.ProblemDetails{
			Status: http.StatusUnprocessableEntity,
			Detail: err.Error(),
		})
	}
	if res, ok := errors.AsType[*awshttp.ResponseError](err); ok {
		return errors.Join(err, &api.ProblemDetails{
			Status: res.HTTPStatusCode(),
			Detail: err.Error(),
		})
	}
	return err
}
