{{- define "argyll.gatewayConfig" -}}
{{- $engine := include "argyll.name" (dict "root" . "component" "engine") -}}
{{- $raft := printf "%s-raft" $engine -}}
{{- $domain := printf "%s.svc.%s" .Release.Namespace .Values.clusterDomain -}}
global
    log stdout format raw local0

# Re-resolve pod names so rescheduled pods are picked up
resolvers cluster
    parse-resolv-conf

defaults
    log global
    mode http
    # Logs the path in place of the request line, and only /callbacks/ for callbacks, whose
    # paths carry a secret
    log-format "%ci:%cp [%tr] %ft %b/%s %TR/%Tw/%Tc/%Tr/%Ta %ST %B %CC %CS %tsc %ac/%fc/%bc/%sc/%rc %sq/%bq %hr %hs \"%HM %[var(txn.logpath)]\""
    option dontlognull
    # A refused connection, such as from an engine that is shutting down, retries on another server
    retries 3
    option redispatch
    timeout connect 5s
    timeout client 300s
    timeout server 300s
    timeout tunnel 1h
    default-server resolvers cluster init-addr last,libc,none

frontend argyll
    bind *:8080
    option forwardfor
    # A TLS load balancer in front already says how the request arrived
    http-request set-header X-Forwarded-Proto http unless { req.hdr(x-forwarded-proto) -m found }
    http-request set-header X-Forwarded-Host %[req.hdr(host)] unless { req.hdr(x-forwarded-host) -m found }
    http-request deny deny_status 404 if { path /ready }

    acl write_callback path_beg /callbacks/
    http-request set-var(txn.logpath) path
    http-request set-var(txn.logpath) str(/callbacks/) if write_callback
    acl write_method method POST PUT DELETE PATCH
    acl write_catalog path_beg /engine/steps /engine/spaces /engine/feeds
    acl write_flow path -i /engine/flows
{{- if and .Values.auth.enabled .Values.keycloak.enabled }}

    acl auth path_beg /auth/
    # Keycloak administration stays inside the cluster
    acl auth_admin path_beg /auth/admin /auth/realms/master
    http-request deny deny_status 404 if auth_admin
    use_backend keycloak if auth
{{- end }}

    acl chat path /chat
    acl health path /health
    acl engine path_beg /engine /callbacks/

    use_backend argyll_chat if chat
    use_backend argyll_health if health
    use_backend argyll_write if write_callback
    use_backend argyll_write if write_method write_catalog
    use_backend argyll_write if write_method write_flow
    use_backend argyll_read if engine
    default_backend argyll_ui
{{- if and .Values.auth.enabled .Values.keycloak.enabled }}

backend keycloak
    server keycloak {{ include "argyll.name" (dict "root" . "component" "keycloak") }}.{{ $domain }}:8080
{{- end }}

backend argyll_chat
    server chat {{ include "argyll.name" (dict "root" . "component" "chat") }}.{{ $domain }}:8072

backend argyll_ui
    server ui {{ include "argyll.name" (dict "root" . "component" "ui") }}.{{ $domain }}:3001

# Keep liveness available during leader elections
backend argyll_health
    option httpchk GET /health
    http-check expect status 200
    default-server inter 2s fastinter 1s downinter 1s rise 1 fall 2
{{- range $i := until (int .Values.engine.replicas) }}
    server {{ $engine }}-{{ $i }} {{ $engine }}-{{ $i }}.{{ $raft }}.{{ $domain }}:8080 check
{{- end }}

backend argyll_read
    option httpchk GET /ready
    http-check expect status 200
    default-server inter 2s fastinter 1s downinter 1s rise 1 fall 2
{{- range $i := until (int .Values.engine.replicas) }}
    server {{ $engine }}-{{ $i }} {{ $engine }}-{{ $i }}.{{ $raft }}.{{ $domain }}:8080 check
{{- end }}

backend argyll_leader
    option httpchk GET /ready
    http-check expect status 200
    http-check expect hdr name X-Argyll-Raft-State value leader
    default-server inter 2s fastinter 1s downinter 1s rise 1 fall 1
{{- range $i := until (int .Values.engine.replicas) }}
    server {{ $engine }}-{{ $i }} {{ $engine }}-{{ $i }}.{{ $raft }}.{{ $domain }}:8080 check
{{- end }}

backend argyll_write
    # Followers can forward proposals to the Raft leader
    option allbackups
    option redispatch 1
{{- range $i := until (int .Values.engine.replicas) }}
    server {{ $engine }}-{{ $i }} {{ $engine }}-{{ $i }}.{{ $raft }}.{{ $domain }}:8080 track argyll_leader/{{ $engine }}-{{ $i }}
    server {{ $engine }}-{{ $i }}-backup {{ $engine }}-{{ $i }}.{{ $raft }}.{{ $domain }}:8080 backup track argyll_read/{{ $engine }}-{{ $i }}
{{- end }}
{{- end -}}
