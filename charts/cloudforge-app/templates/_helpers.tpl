{{- define "cloudforge-app.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "cloudforge-app.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- include "cloudforge-app.name" . }}
{{- end }}
{{- end }}

{{- define "cloudforge-app.labels" -}}
app.kubernetes.io/name: {{ include "cloudforge-app.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
cloudforge.io/managed: "true"
{{- end }}

{{- define "cloudforge-app.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cloudforge-app.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "cloudforge-app.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "cloudforge-app.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
