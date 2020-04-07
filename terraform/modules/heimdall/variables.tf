variable "name" {
  type        = string
  description = "Name used to identify resources"
}

variable "container_tag" {
  type        = string
  description = "Tag of the heimdall container in the registry to be used"
  default     = "latest"
}

variable "container_count" {
  type        = number
  description = "The number of containers to deploy in the heimdall service"
  default     = 1
}

variable "cluster_id" {
  type        = string
  description = "ID of the ECS cluster that the heimdall service will run in"
}

variable "external_lb_subnets" {
  type        = list(string)
  description = "VPC subnets for the external heimdall service load balancers"
}

variable "internal_lb_subnets" {
  type        = list(string)
  description = "VPC subnets for the internal heimdall service load balancers"
}

variable "vpc_id" {
  type        = string
  description = "VPC ID for the heimdall service load balancers"
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

variable "endpoints" {
  type        = map(string)
  description = "Endpoints for accessing backend services {'service-name' = 'endpoint'}"
}
