# Deploy

The Helm chart in `helm/argyll` deploys Argyll to Kubernetes. The [Deploy on Kubernetes](https://www.argyll.app/docs/getting-started/kubernetes/) guide covers installing it. The Docker Compose stack at the repository root is for local development.

## Build the Images

The chart pulls these images from `ghcr.io/kode4food`, tagged with the chart's `appVersion`:

| Image | Built from |
|---|---|
| `argyll-engine` | `engine/Dockerfile` |
| `argyll-ui` | `web/Dockerfile` |
| `argyll-chat` | `chat/Dockerfile` |
| `argyll-mcp` | `mcp/Dockerfile` |
| `argyll-keycloak` | `keycloak/Dockerfile` |
| `argyll-agent-anthropic`, `argyll-agent-bedrock`, `argyll-agent-gemini`, `argyll-agent-openai` | `workers/agent/Dockerfile` with the `WORKER` build argument |

Cluster nodes usually run `linux/amd64`, so build for both architectures. From the repository root, signed in to the registry:

```bash
REGISTRY=ghcr.io/kode4food
TAG=0.1.0
PLATFORMS=linux/amd64,linux/arm64

build() { docker buildx build --platform "$PLATFORMS" --push -t "$REGISTRY/argyll-$1:$TAG" "${@:2}"; }

build engine -f engine/Dockerfile .
build chat -f chat/Dockerfile .
build mcp -f mcp/Dockerfile .
build ui web
build keycloak keycloak
for p in anthropic bedrock gemini openai; do
  build "agent-$p" --build-arg "WORKER=argyll-agent-$p" -f workers/agent/Dockerfile .
done
```

Set `TAG` to the chart's `appVersion`, or install the chart with `image.tag` set to your tag.

## Check the Chart

```bash
helm lint deploy/helm/argyll --set publicUrl=https://argyll.example.com
helm template argyll deploy/helm/argyll --set publicUrl=https://argyll.example.com
```

After installing, `helm test argyll -n argyll --logs` checks routing, sign-in, the paths between components, and that every component other than the gateway is private.
