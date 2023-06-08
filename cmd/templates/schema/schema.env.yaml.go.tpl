schema:
  format:
    openapi: "3.0.0"
  environment_input_type: environment
  types:
    environment:
      type: object
      description: Environment input properties
      properties:
      {{ range $v := . }}{{ if ne $v.Name "name" }}
        {{ $v.Name }}:
          title: {{ $v.Name }}
          type: {{ $v.Type }}
          {{ if ne "" $v.ArrayType }}items:
            type: {{ $v.ArrayType }}
          {{ end }}
          description: "{{ $v.Description }}"
          {{ if ne nil $v.Default }}default: {{ $v.Default }}{{ end }}
      {{ end }}{{ end }}
