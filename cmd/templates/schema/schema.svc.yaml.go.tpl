schema:
  format:
    openapi: "3.0.0"
  service_input_type: service
  types:
    service:
      type: object
      description: Service input properties
      properties:
      {{ range $v := . }}{{ if and (ne $v.Name "name") (ne $v.Name "environment") }}
        {{ $v.Name }}:
          title: {{ $v.Name }}
          type: {{ $v.Type }}
          {{ if ne "" $v.ArrayType }}items:
            type: {{ $v.ArrayType }}
          {{ end }}
          description: "{{ $v.Description }}"
          {{ if ne nil $v.Default }}default: {{ $v.Default }}{{ end }}
      {{ end }}{{ end }}
