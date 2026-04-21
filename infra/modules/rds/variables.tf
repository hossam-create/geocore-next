variable "identifier" {
  description = "RDS instance identifier"
  type        = string
}

variable "engine_version" {
  description = "PostgreSQL engine version"
  type        = string
  default     = "16"
}

variable "instance_class" {
  description = "RDS instance type"
  type        = string
  default     = "db.t4g.medium"
}

variable "allocated_storage" {
  description = "Initial storage in GB"
  type        = number
  default     = 100
}

variable "max_allocated_storage" {
  description = "Max storage for auto-scaling (GB)"
  type        = number
  default     = 500
}

variable "db_name" {
  description = "Database name"
  type        = string
  default     = "geocore"
}

variable "username" {
  description = "Master username"
  type        = string
  default     = "geocore"
}

variable "password" {
  description = "Master password (from SSM/Vault)"
  type        = string
  sensitive   = true
}

variable "multi_az" {
  description = "Enable multi-AZ for HA"
  type        = bool
  default     = true
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

variable "subnet_group_name" {
  description = "DB subnet group name"
  type        = string
}

variable "backup_retention_days" {
  description = "Backup retention period"
  type        = number
  default     = 7
}

variable "deletion_protection" {
  description = "Prevent accidental deletion"
  type        = bool
  default     = true
}

variable "skip_final_snapshot" {
  description = "Skip final snapshot on destroy (dev only)"
  type        = bool
  default     = false
}

variable "performance_insights" {
  description = "Enable Performance Insights"
  type        = bool
  default     = true
}

variable "tags" {
  description = "Common tags"
  type        = map(string)
  default     = {}
}
