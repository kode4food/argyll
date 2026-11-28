module github.com/kode4food/argyll/workers/agent

go 1.27.0

require (
	github.com/anthropics/anthropic-sdk-go v1.84.0
	github.com/aws/aws-sdk-go-v2 v1.48.0
	github.com/aws/aws-sdk-go-v2/config v1.33.21
	github.com/aws/aws-sdk-go-v2/service/bedrock v1.75.0
	github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.64.0
	github.com/kode4food/argex v0.1.11
	github.com/kode4food/argyll/chat v0.0.0
	github.com/kode4food/argyll/engine v0.0.0-20261124113524-00b2ef03c42f
	github.com/kode4food/lru v0.0.0-20261011080515-d1c90c62daee
	github.com/modelcontextprotocol/go-sdk v1.8.0
	github.com/openai/openai-go/v3 v3.106.0
	github.com/stretchr/testify v1.12.1
	golang.org/x/oauth2 v0.38.0
	google.golang.org/genai v1.79.1
)

require (
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/auth v0.24.4 // indirect
	cloud.google.com/go/compute/metadata v0.10.0 // indirect
	github.com/Shopify/go-lua v0.0.0-20261007132305-3f823f845ff9 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.27 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.21.3 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.20.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.5.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.25 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.14.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.14.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.39.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.44.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.53.0 // indirect
	github.com/aws/smithy-go v1.28.9 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.6.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/coder/websocket v1.8.15 // indirect
	github.com/coreos/go-oidc/v3 v3.21.0 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/go-jose/go-jose/v4 v4.1.9 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/google/s2a-go v0.1.11 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.25 // indirect
	github.com/googleapis/gax-go/v2 v2.27.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/invopop/jsonschema v0.14.0 // indirect
	github.com/kode4food/timebox v0.5.1-0.20261128092604-3d8e4cba7996 // indirect
	github.com/pb33f/go-yaml v0.2.0 // indirect
	github.com/pb33f/ordered-map/v2 v2.3.2 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/standard-webhooks/standard-webhooks/libraries v1.3.0 // indirect
	github.com/tidwall/gjson v1.20.6 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.3.1 // indirect
	github.com/tidwall/sjson v1.2.11 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.2 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.72.0 // indirect
	go.opentelemetry.io/otel v1.47.0 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	go.opentelemetry.io/otel/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/trace v1.47.0 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
	golang.org/x/crypto v0.59.0 // indirect
	golang.org/x/net v0.63.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.44.0 // indirect
	golang.org/x/time v0.16.0 // indirect
	google.golang.org/api v0.316.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261123191846-1e707ab9fbc2 // indirect
	google.golang.org/grpc v1.85.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/kode4food/argyll/engine => ../../engine

replace github.com/kode4food/argyll/chat => ../../chat
