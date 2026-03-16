variable "region" {
  description = "The region"
  type        = string
}

variable "service_name" {
  description = "The name of the Cloud Run service"
  type        = string
}

variable "image_url" {
  description = "The URL of the container image to deploy"
  type        = string
}

variable "env_vars" {
  description = "Map of environment variables to inject into the container"
  type        = map(string)
  default     = {}
}

variable "network_name" {
  description = "The name of the VPC network for Direct VPC Egress"
  type        = string
}

variable "subnet_name" {
  description = "The name of the subnetwork for Direct VPC Egress"
  type        = string
}

variable "service_account_email" {
  description = "The email of the service account to run the service as"
  type        = string
}

variable "cpu" {
  description = "CPU limit for the Cloud Run container"
  type        = string
}

variable "memory" {
  description = "Memory limit for the Cloud Run container"
  type        = string
}

variable "ingress" {
  description = "Ingress traffic configuration for the Cloud Run service"
  type        = string
}

variable "vpc_egress" {
  description = "VPC egress traffic configuration for the Cloud Run service"
  type        = string
}

variable "allow_unauthenticated" {
  description = "Whether to allow unauthenticated access to the service"
  type        = bool
}
