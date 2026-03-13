variable "allowed_regions" {
  type        = list(string)
  description = "List of regions to allow in the Geo-Location filter"
}

variable "rate_limit_count" {
  type        = number
  description = "The threshold for IP-based rate limiting"
}

variable "app_name" {
  type        = string
  description = "The application name"
}
