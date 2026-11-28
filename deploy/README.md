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

## Try It on kind

[kind](https://kind.sigs.k8s.io/) runs a cluster inside Docker. Build the images locally and load them straight into the cluster. From the repository root:

```bash
kind create cluster --name argyll
```

```bash
TAG=dev
build() { docker build -q -t "ghcr.io/kode4food/argyll-$1:$TAG" "${@:2}"; }

build engine -f engine/Dockerfile .
build chat -f chat/Dockerfile .
build mcp -f mcp/Dockerfile .
build ui web
build keycloak keycloak
for i in engine chat mcp ui keycloak; do
  kind load docker-image "ghcr.io/kode4food/argyll-$i:$TAG" --name argyll
done
```

```bash
helm install argyll deploy/helm/argyll -n argyll --create-namespace --set publicUrl=http://localhost:8090 --set image.tag=dev --set ingress.enabled=false --wait --timeout 10m
```

The install prints the `kubectl port-forward` commands for the gateway and the Keycloak admin console, and how to read both passwords. Forward the gateway, then open `http://localhost:8090`.

Browsers sign in only from a secure origin, so `publicUrl` must use HTTPS or `localhost`.

When `publicUrl` points to loopback, the engine gives workers a callback URL on the gateway's Kubernetes Service. If async HTTP Steps run outside the cluster, set `engine.webhookBaseUrl` to a gateway address reachable from all workers and async Steps.

Keycloak keeps the `publicUrl`, `keycloak.adminPort`, `auth.audience`, `auth.clientId`, and `auth.rolesClaim` it was installed with in its realm settings. To change any of them on kind, run `kubectl delete namespace argyll` and install again.

## Check the Chart

```bash
helm lint deploy/helm/argyll --set publicUrl=https://argyll.example.com
helm template argyll deploy/helm/argyll --set publicUrl=https://argyll.example.com
```

After installing, `helm test argyll -n argyll --logs` checks routing, sign-in, the paths between components, and that every component other than the gateway is private.
