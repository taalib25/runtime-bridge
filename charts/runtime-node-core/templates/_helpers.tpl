{{- define "runtime-node-core.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "runtime-node-core.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := include "runtime-node-core.name" . -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "runtime-node-core.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels.
Backend-owned commonLabels (hermescloud.dev/*) are emitted FIRST so the
chart/bridge-owned standard labels below always win on any key collision
(YAML last-key-wins) — commonLabels can never override selectorLabels.
See docs/label-contract.md.
*/}}
{{- define "runtime-node-core.labels" -}}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
helm.sh/chart: {{ include "runtime-node-core.chart" . }}
{{ include "runtime-node-core.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: runtime-node-core
{{- end -}}

{{/*
Selector labels — immutable Kubernetes identity only. Never include business
metadata or commonLabels here; this set is used in immutable Deployment selectors.
*/}}
{{- define "runtime-node-core.selectorLabels" -}}
app.kubernetes.io/name: {{ include "runtime-node-core.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Common annotations — backend-owned commonAnnotations (hermescloud.dev/*).
Emits nothing when unset, so callers guard with `with`.
*/}}
{{- define "runtime-node-core.annotations" -}}
{{- with .Values.commonAnnotations }}
{{ toYaml . }}
{{- end }}
{{- end -}}

{{- define "runtime-node-core.image" -}}
{{- if .Values.image.digest -}}
{{ .Values.image.repository }}@{{ .Values.image.digest }}
{{- else -}}
{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}
{{- end -}}
{{- end -}}

{{- define "runtime-node-core.secretName" -}}
{{- if .Values.secrets.existingSecret -}}
{{- .Values.secrets.existingSecret -}}
{{- else -}}
{{- printf "%s-secrets" (include "runtime-node-core.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
