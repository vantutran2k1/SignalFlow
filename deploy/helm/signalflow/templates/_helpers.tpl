{{/*
Expand the name of the chart.
*/}}
{{- define "signalflow.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Fully qualified app name. Truncated to 63 chars (RFC 1123 label limit).
*/}}
{{- define "signalflow.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Bundled Postgres resources share this prefix.
*/}}
{{- define "signalflow.postgresFullname" -}}
{{- printf "%s-postgres" (include "signalflow.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Chart name and version for app.kubernetes.io/* labels.
*/}}
{{- define "signalflow.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Full set of common labels.
*/}}
{{- define "signalflow.labels" -}}
helm.sh/chart: {{ include "signalflow.chart" . }}
{{ include "signalflow.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels — stable across upgrades (must not include version).
*/}}
{{- define "signalflow.selectorLabels" -}}
app.kubernetes.io/name: {{ include "signalflow.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
ServiceAccount name.
*/}}
{{- define "signalflow.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "signalflow.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/*
Image reference — prefers explicit tag, falls back to Chart.AppVersion.
*/}}
{{- define "signalflow.image" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end -}}

{{/*
Computed DATABASE_URL. If the bundled Postgres is enabled and the user
didn't override secrets.DATABASE_URL, point at the bundled Postgres service.
Otherwise use the value as provided.
*/}}
{{- define "signalflow.databaseURL" -}}
{{- if and .Values.postgres.enabled (not .Values.secrets.DATABASE_URL) -}}
{{- $auth := .Values.postgres.auth -}}
{{- $host := include "signalflow.postgresFullname" . -}}
{{- printf "postgres://%s:%s@%s:5432/%s?sslmode=disable" $auth.username $auth.password $host $auth.database -}}
{{- else -}}
{{- .Values.secrets.DATABASE_URL -}}
{{- end -}}
{{- end -}}
