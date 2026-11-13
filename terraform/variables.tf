variable "region" {
  description = "GCP deployment region"
  type        = string
  default     = "asia-southeast1"
}

variable "projec_id" {
  description = "GCP project ID"
  type        = string
}

variable "admin_cidr" {
  description = "Public IPv4 CIDR allowed to access the GKE API"
  type        = string
}
