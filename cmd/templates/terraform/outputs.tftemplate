{{$moduleName := .ModuleName}}
{{ range $o := .Outputs }}
output "{{ $o.Name }}" {
  description = "{{ $o.Description }}"
  value       = module.{{ $moduleName }}.{{ $o.Name }}
}
{{ end }}
