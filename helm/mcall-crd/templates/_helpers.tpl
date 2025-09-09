{{/*
Expand the name of the chart.
*/}}
{{- define "mcall-crd.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
Environment-specific naming is applied.
*/}}
{{- define "mcall-crd.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $envSuffix := .Values.environment.suffix | default "" }}
{{- $envPrefix := .Values.environment.prefix | default "" }}
{{- printf "%s%s%s" $envPrefix .Release.Name $envSuffix | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "mcall-crd.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "mcall-crd.labels" -}}
helm.sh/chart: {{ include "mcall-crd.chart" . }}
{{ include "mcall-crd.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "mcall-crd.selectorLabels" -}}
app.kubernetes.io/name: {{ include "mcall-crd.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "mcall-crd.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "mcall-crd.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the namespace name
*/}}
{{- define "mcall-crd.namespace" -}}
{{- if .Values.namespace.create }}
{{- .Values.namespace.name }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Create the image name
*/}}
{{- define "mcall-crd.image" -}}
{{- $registry := .Values.global.imageRegistry | default .Values.image.registry }}
{{- $tag := .Values.image.tag | default .Chart.AppVersion | toString }}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry .Values.image.repository $tag }}
{{- else }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}
{{- end }}

{{/*
Create the image pull secrets
*/}}
{{- define "mcall-crd.imagePullSecrets" -}}
{{- if .Values.global.imagePullSecrets }}
imagePullSecrets:
{{- range .Values.global.imagePullSecrets }}
  - name: {{ . }}
{{- end }}
{{- else if .Values.imagePullSecrets }}
imagePullSecrets:
{{- range .Values.imagePullSecrets }}
  - name: {{ . }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create the resource requirements
*/}}
{{- define "mcall-crd.resources" -}}
{{- if .Values.controller.resources }}
resources:
{{- toYaml .Values.controller.resources | nindent 2 }}
{{- end }}
{{- end }}

{{/*
Create the node selector
*/}}
{{- define "mcall-crd.nodeSelector" -}}
{{- if .Values.controller.nodeSelector }}
nodeSelector:
{{- toYaml .Values.controller.nodeSelector | nindent 2 }}
{{- end }}
{{- end }}

{{/*
Create the tolerations
*/}}
{{- define "mcall-crd.tolerations" -}}
{{- if .Values.controller.tolerations }}
tolerations:
{{- toYaml .Values.controller.tolerations | nindent 2 }}
{{- end }}
{{- end }}

{{/*
Create the affinity
*/}}
{{- define "mcall-crd.affinity" -}}
{{- if .Values.controller.affinity }}
affinity:
{{- toYaml .Values.controller.affinity | nindent 2 }}
{{- end }}
{{- end }}

{{/*
Create the environment variables
*/}}
{{- define "mcall-crd.env" -}}
{{- if .Values.controller.env }}
env:
{{- range $key, $value := .Values.controller.env }}
- name: {{ $key }}
  value: {{ $value | quote }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create the webhook certificate secret name
*/}}
{{- define "mcall-crd.webhookCertSecretName" -}}
{{- printf "%s-webhook-certs" (include "mcall-crd.fullname" .) }}
{{- end }}

{{/*
Create the webhook service name
*/}}
{{- define "mcall-crd.webhookServiceName" -}}
{{- printf "%s-webhook-service" (include "mcall-crd.fullname" .) }}
{{- end }}

{{/*
Create the metrics service name
*/}}
{{- define "mcall-crd.metricsServiceName" -}}
{{- printf "%s-metrics" (include "mcall-crd.fullname" .) }}
{{- end }}

{{/*
Create the liveness probe
*/}}
{{- define "mcall-crd.livenessProbe" -}}
{{- if .Values.controller.livenessProbe }}
livenessProbe:
{{- toYaml .Values.controller.livenessProbe | nindent 2 }}
{{- end }}
{{- end }}

{{/*
Create the readiness probe
*/}}
{{- define "mcall-crd.readinessProbe" -}}
{{- if .Values.controller.readinessProbe }}
readinessProbe:
{{- toYaml .Values.controller.readinessProbe | nindent 2 }}
{{- end }}
{{- end }}


