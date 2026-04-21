variable "cluster_id" {
  description = "Redis cluster ID"
  type        = string
}

variable "engine_version" {
  description = "Redis engine version"
  type        = string
  default     = "7.1"
}

variable "node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.t3.micro"
}

variable "num_cache_nodes" {
  description = "Number of cache nodes (2+ enables failover)"
  type        = number
  default     = 2
}

variable "parameter_group_name" {
  description = "Parameter group name"
  type        = string
  default     = "default.redis7"
}

variable "subnet_group_name" {
  description = "ElastiCache subnet group name"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "vpc_cidr" {
  description = "VPC CIDR for SG rules"
  type        = string
  default     = "10.0.0.0/16"
}

variable "auth_token" {
  description = "Redis AUTH token (empty = no auth)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "multi_az" {
  description = "Enable Multi-AZ"
  type        = bool
  default     = false
}

variable "snapshot_retention_days" {
  description = "Snapshot retention days"
  type        = number
  default     = 1
}

variable "tags" {
  description = "Common tags"
  type        = map(string)
  default     = {}
}
