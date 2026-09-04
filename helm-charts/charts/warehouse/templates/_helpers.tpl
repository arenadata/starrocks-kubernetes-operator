{{- define "starrockswarehouse.name" -}}
{{ .Release.Name }}
{{- end }}

{{- define "starrockswarehouse.namespace" -}}
{{ .Release.Namespace }}
{{- end }}

{{- define "starrockswarehouse.labels" -}}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "starrockswarehouse.config.secret.name" -}}
{{- print (include "starrockswarehouse.name" .) "-conf" }}
{{- end }}

{{/*
The directory the compute nodes read their configuration from. It has to match the one the operator
derives from STARROCKS_ROOT, otherwise the operator can not read the ports out of the configuration.
*/}}
{{- define "starrockswarehouse.conf.dir" -}}
{{- print "/opt/starrocks/cn/conf" }}
{{- end }}

{{- define "starrockswarehouse.config" -}}
cn.conf: |
{{- if .Values.spec.config }}
{{- .Values.spec.config | nindent 2 }}
{{- end }}
{{- end }}

{{/*
starrockswarehouse.config.hash is used to calculate the hash value of the cn.conf, and due to the length limit, only
the first 8 digits are taken, which will be used as the annotations for pods.
*/}}
{{- define "starrockswarehouse.config.hash" }}
  {{- if .Values.spec.config }}
    {{- $hash := toJson .Values.spec.config | sha256sum | trunc 8 }}
    {{- printf "%s" $hash }}
  {{- else }}
    {{- printf "no-config" }}
  {{- end }}
{{- end }}

{{- define "starrockswarehouse.webserver.port" -}}
{{- include "starrockswarehouse.get.webserver.port" .Values.spec }}
{{- end }}

{{- define "starrockswarehouse.get.webserver.port" -}}
{{- $config := index .config  -}}
{{- $configMap := dict -}}
{{- range $line := splitList "\n" $config -}}
{{- $pair := splitList "=" $line -}}
{{- if eq (len $pair) 2 -}}
{{- $_ := set $configMap (trim (index $pair 0)) (trim (index $pair 1)) -}}
{{- end -}}
{{- end -}}
{{- if (index $configMap "webserver_port") -}}
{{- print (index $configMap "webserver_port") }}
{{- end }}
{{- end }}

{{- define "starrockscluster.cn.data.suffix" -}}
{{- print "-data" }}
{{- end }}

{{- define "starrockscluster.cn.data.path" -}}
{{- print "/opt/starrocks/cn/storage" }}
{{- end }}

{{- define "starrockscluster.cn.log.suffix" -}}
{{- print "-log" }}
{{- end }}

{{- define "starrockscluster.cn.log.path" -}}
{{- print "/opt/starrocks/cn/log" }}
{{- end }}
