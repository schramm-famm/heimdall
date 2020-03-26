variable "name" {
  type        = string
  description = "Name used to identify resources"
}

variable "container_tag" {
  type        = string
  description = "Tag of the heimdall container in the registry to be used"
  default     = "latest"
}

variable "cluster_id" {
  type        = string
  description = "ID of the ECS cluster that the heimdall service will run in"
}

variable "private_key" {
  type        = string
  description = "Path to the private RSA key"
  default     = "id_rsa"
}
