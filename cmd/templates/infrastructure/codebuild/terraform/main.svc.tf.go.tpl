terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }

  backend "s3" {}
}

provider "aws" {
  default_tags {
    tags = {
      "proton:environment"      = var.environment.name
      "proton:service"          = var.service.name,
      "proton:service_instance" = var.service_instance.name,
    }
  }
}

module "{{ .ModuleName }}" {
  source = "./src"

{{ range $v := .Variables }}
  {{ if eq $v.Name "name" }}name = "${var.service.name}-${var.service_instance.name}"{{ else if eq $v.Name "environment" }}environment = var.environment.name{{ else }}
  {{ $v.Name }} = var.service_instance.inputs.{{ $v.Name }}
{{ end }}{{ end }}
}
