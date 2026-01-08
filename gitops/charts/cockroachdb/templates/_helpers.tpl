{{/*
Expand the name of the chart.
*/}}
{{- define "cockroachdb.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cockroachdb.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "cockroachdb.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "cockroachdb.labels" -}}
helm.sh/chart: {{ include "cockroachdb.chart" . }}
{{ include "cockroachdb.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "cockroachdb.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cockroachdb.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "cockroachdb.serviceAccountName" -}}
{{- if .Values.statefulset.serviceAccount.create }}
{{- default (include "cockroachdb.fullname" .) .Values.statefulset.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.statefulset.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Generate join addresses for CockroachDB
For multi-region: includes both local and remote region nodes
Format: local-0.local,local-1.local,remote-0.remote:26257,...
*/}}
{{- define "cockroachdb.joinAddresses" -}}
{{- $replicas := .Values.statefulset.replicas | int }}
{{- $name := include "cockroachdb.fullname" . }}
{{- $addresses := list }}
{{- $region := .Values.region.name }}
{{- $remoteRegion := .Values.region.remoteRegion | default "" }}
{{- $remoteJoinAddresses := .Values.region.remoteJoinAddresses | default "" }}
{{- $useLoadBalancer := .Values.region.useLoadBalancerForJoin | default false }}
{{- $serviceName := include "cockroachdb.fullname" . }}

{{- /* Add local region nodes */}}
{{- range $i := until $replicas }}
  {{- if $useLoadBalancer }}
    {{- /* Use LoadBalancer service for cross-cluster communication */}}
    {{- $addresses = append $addresses (printf "%s.%s.svc.cluster.local:26257" $serviceName (include "cockroachdb.namespace" .)) }}
  {{- else }}
    {{- /* Use pod DNS for same-cluster communication */}}
    {{- $addresses = append $addresses (printf "%s-%d.%s:26257" $name $i $name) }}
  {{- end }}
{{- end }}

{{- /* Add remote region nodes if configured */}}
{{- if $remoteJoinAddresses }}
  {{- $remoteAddresses := splitList "," $remoteJoinAddresses }}
  {{- range $addr := $remoteAddresses }}
    {{- $addresses = append $addresses (trim $addr) }}
  {{- end }}
{{- end }}

{{- join "," $addresses }}
{{- end }}

{{/*
Get namespace (defaults to release namespace)
*/}}
{{- define "cockroachdb.namespace" -}}
{{- .Release.Namespace | default "default" }}
{{- end }}
