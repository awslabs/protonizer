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

# TODO: this is just an example resource
# replace this with your real template resources here
resource "aws_cloudwatch_log_group" "example" {
  name = var.service_instance.inputs.{{.Name}}
}
