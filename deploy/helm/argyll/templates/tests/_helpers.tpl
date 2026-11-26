{{/* A test pod running a shell script. Tests reach Argyll only through the gateway, like any client */}}
{{- define "argyll.testPod" -}}
{{- $ctx := dict "root" .root "component" (printf "test-%s" .name) -}}
apiVersion: v1
kind: Pod
metadata:
  name: {{ include "argyll.name" $ctx }}
  labels:
    {{- include "argyll.labels" $ctx | nindent 4 }}
  annotations:
    helm.sh/hook: test
    helm.sh/hook-delete-policy: before-hook-creation
spec:
  restartPolicy: Never
  {{- include "argyll.podDefaults" .root | nindent 2 }}
  containers:
    - name: test
      image: {{ include "argyll.image" (dict "root" .root "name" "engine") }}
      imagePullPolicy: {{ .root.Values.image.pullPolicy }}
      {{- with .env }}
      env:
        {{- . | nindent 8 }}
      {{- end }}
      command:
        - sh
        - -c
        - |
          fail=0
          gateway=http://{{ include "argyll.name" (dict "root" .root "component" "gateway") }}:8080
          code() { curl -s -o /dev/null -m 10 -w '%{http_code}' "$@"; }
          expect() {
            if [ "$3" = "$2" ]; then echo "ok    $1"; else echo "FAIL  $1: expected $2, got $3"; fail=1; fi
          }
          has() {
            case "$3" in *"$2"*) echo "ok    $1";; *) echo "FAIL  $1: missing $2"; fail=1;; esac
          }
          lacks() {
            case "$3" in *"$2"*) echo "FAIL  $1: found $2"; fail=1;; *) echo "ok    $1";; esac
          }
          {{- .script | nindent 10 }}
          exit $fail
{{- end -}}
