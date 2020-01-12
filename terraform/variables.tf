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

variable "key_name" {
  type        = string
  description = "AWS key pair name for SSH access to EC2 instances"
}
