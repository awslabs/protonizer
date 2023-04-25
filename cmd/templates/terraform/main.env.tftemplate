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
      "proton:environment" = var.environment.name
    }
  }
}

module "{{ .ModuleName }}" {
  source = "./src"

{{ range $v := .Variables }}
  {{ if eq $v.Name "name" }}name = var.environment.name{{ else }}
  {{ $v.Name }} = var.environment.inputs.{{ $v.Name }}
{{ end }}{{ end }}
}
