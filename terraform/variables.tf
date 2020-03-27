variable "name" {
  type        = string
  description = "Name used to identify resources"
}

variable "access_key" {
  type        = string
  description = "AWS access key ID"
}

variable "secret_key" {
  type        = string
  description = "AWS secret access key"
}

variable "region" {
  type        = string
  description = "AWS region to deploy where resources will be deployed"
  default     = "us-east-2"
}

variable "container_tag" {
  type        = string
  description = "Tag of the heimdall container in the registry to be used"
  default     = "latest"
}

variable "private_key_jwt" {
  type        = string
  description = "Path to the private RSA key in the container for the JWT"
  default     = "id_rsa"
}

variable "private_key_cert" {
  type        = string
  description = "Local path to the private RSA key for the TLS certificate"
}

variable "cert" {
  type        = string
  description = "Local path to the TLS certificate"
}
