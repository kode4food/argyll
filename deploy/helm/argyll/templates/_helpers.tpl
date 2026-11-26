{{- define "argyll.fullname" -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "argyll.name" -}}
{{- printf "%s-%s" (include "argyll.fullname" .root) .component | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "argyll.selectorLabels" -}}
app.kubernetes.io/name: {{ .root.Chart.Name }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end -}}

{{- define "argyll.labels" -}}
{{ include "argyll.selectorLabels" . }}
app.kubernetes.io/version: {{ .root.Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .root.Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .root.Chart.Name .root.Chart.Version }}
{{- end -}}

{{- define "argyll.image" -}}
{{- printf "%s/argyll-%s:%s" .root.Values.image.registry .name (.root.Values.image.tag | default .root.Chart.AppVersion) -}}
{{- end -}}

{{- define "argyll.publicUrl" -}}
{{- required "publicUrl is required, such as https://argyll.example.com" .Values.publicUrl | trimSuffix "/" -}}
{{- end -}}

{{- define "argyll.publicHost" -}}
{{- (urlParse (include "argyll.publicUrl" .)).host | splitList ":" | first -}}
{{- end -}}

{{- define "argyll.keycloakUrl" -}}
{{- printf "http://%s:8080/auth/realms/argyll" (include "argyll.name" (dict "root" . "component" "keycloak")) -}}
{{- end -}}

{{- define "argyll.issuer" -}}
{{- if .Values.keycloak.enabled -}}
{{- printf "%s/auth/realms/argyll" (include "argyll.publicUrl" .) -}}
{{- else -}}
{{- required "auth.external.issuer is required when keycloak.enabled is false" .Values.auth.external.issuer -}}
{{- end -}}
{{- end -}}

{{- define "argyll.jwksUrl" -}}
{{- if .Values.keycloak.enabled -}}
{{- printf "%s/protocol/openid-connect/certs" (include "argyll.keycloakUrl" .) -}}
{{- else -}}
{{- .Values.auth.external.jwksUrl -}}
{{- end -}}
{{- end -}}

{{- define "argyll.tokenUrl" -}}
{{- if .Values.keycloak.enabled -}}
{{- printf "%s/protocol/openid-connect/token" (include "argyll.keycloakUrl" .) -}}
{{- else -}}
{{- required "auth.external.tokenUrl is required when keycloak.enabled is false" .Values.auth.external.tokenUrl -}}
{{- end -}}
{{- end -}}

{{- define "argyll.authority" -}}
{{- if .Values.keycloak.enabled -}}
{{- include "argyll.issuer" . -}}
{{- else -}}
{{- .Values.auth.external.authority | default .Values.auth.external.issuer -}}
{{- end -}}
{{- end -}}

{{- define "argyll.keycloakSecret" -}}
{{- .Values.keycloak.existingSecret | default (include "argyll.name" (dict "root" . "component" "keycloak")) -}}
{{- end -}}

{{/* Sign-in settings shared by the engine, chat, and agent workers */}}
{{- define "argyll.authEnv" -}}
{{- if .Values.auth.enabled }}
- name: AUTH_PROVIDER
  value: oidc
- name: AUTH_ISSUER
  value: {{ include "argyll.issuer" . | quote }}
- name: AUTH_AUDIENCE
  value: {{ .Values.auth.audience | quote }}
{{- with include "argyll.jwksUrl" . }}
- name: AUTH_JWKS_URL
  value: {{ . | quote }}
{{- end }}
{{- with .Values.auth.rolesClaim }}
- name: AUTH_ROLES_CLAIM
  value: {{ . | quote }}
{{- end }}
{{- else }}
- name: AUTH_PROVIDER
  value: none
{{- end }}
{{- end -}}

{{/* A service account's client credentials, for the engine ("engine") or agent workers ("agent") */}}
{{- define "argyll.accountEnv" -}}
{{- if .root.Values.auth.enabled }}
- name: ARGYLL_TOKEN_URL
  value: {{ include "argyll.tokenUrl" .root | quote }}
{{- if .root.Values.keycloak.enabled }}
- name: ARGYLL_CLIENT_ID
  value: {{ ternary "argyll-engine" "agent-workers" (eq .account "engine") }}
- name: ARGYLL_CLIENT_SECRET
  valueFrom:
    secretKeyRef:
      name: {{ include "argyll.keycloakSecret" .root }}
      key: {{ .account }}-client-secret
{{- else }}
- name: ARGYLL_CLIENT_ID
  valueFrom:
    secretKeyRef:
      name: {{ required "auth.external.existingSecret is required when keycloak.enabled is false" .root.Values.auth.external.existingSecret }}
      key: {{ .account }}-client-id
- name: ARGYLL_CLIENT_SECRET
  valueFrom:
    secretKeyRef:
      name: {{ .root.Values.auth.external.existingSecret }}
      key: {{ .account }}-client-secret
{{- end }}
{{- end }}
{{- end -}}

{{/* Each enabled provider's worker, with the default worker also serving unlisted providers */}}
{{- define "argyll.agentWorkers" -}}
{{- $root := . -}}
{{- $entries := list -}}
{{- range $provider := list "anthropic" "bedrock" "gemini" "openai" -}}
{{- if (index $root.Values.agents $provider).enabled -}}
{{- $url := printf "http://%s:8070" (include "argyll.name" (dict "root" $root "component" (printf "agent-%s" $provider))) -}}
{{- $entries = append $entries (printf "%s=%s" $provider $url) -}}
{{- if eq $provider $root.Values.agents.default -}}
{{- $entries = append $entries (printf "*=%s" $url) -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- join "," $entries -}}
{{- end -}}

{{- define "argyll.raftServers" -}}
{{- $servers := list -}}
{{- range $i := until (int .root.Values.engine.replicas) -}}
{{- $servers = append $servers (printf "%s-%d=%s-%d.%s:9701" $.name $i $.name $i $.raft) -}}
{{- end -}}
{{- join "," $servers -}}
{{- end -}}

{{- define "argyll.podDefaults" -}}
{{- with .Values.imagePullSecrets }}
imagePullSecrets:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- end -}}
