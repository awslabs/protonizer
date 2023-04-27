variable "name" {
  description = "This should be mapped to proton metadata"
  type        = string
}

variable "environment" {
  description = "This should be mapped to proton metadata for services"
  type        = string
}

variable "vpc_cidr" {
  description = "The CIDR range for the VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "private_subnet_one_cidr" {
  description = "The CIDR range for private subnet one"
  type        = string
  default     = "10.0.128.0/18"
}

variable "private_subnet_two_cidr" {
  description = "The CIDR range for private subnet two"
  type        = string
  default     = "10.0.192.0/18"
}

variable "public_subnet_one_cidr" {
  description = "The CIDR range for public subnet one"
  type        = string
  default     = "10.0.0.0/18"
}

variable "public_subnet_two_cidr" {
  description = "The CIDR range for public subnet two"
  type        = string
  default     = "10.0.64.0/18"
}

variable "quote_test" {
  description = "this variable is used to test \"quotes\" in descriptions"
  type        = string
}
