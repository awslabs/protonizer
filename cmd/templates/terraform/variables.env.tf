# required by proton
variable "environment" {
  description = "The Proton Environment"
  type = object({
    name   = string
    inputs = any
  })
  default = null
}
