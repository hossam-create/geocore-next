# ── Global Variables ────────────────────────────────────────────────────────

variable "project" {
  description = "Project name prefix"
  type        = string
  default     = "geocore"
}

variable "environment" {
  description = "Environment name (dev/staging/prod)"
  type        = string

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be dev, staging, or prod."
  }
}

# ── Network ─────────────────────────────────────────────────────────────────

variable "vpc_cidr" {
  type    = string
  default = "10.0.0.0/16"
}

variable "azs" {
  type    = list(string)
  default = ["us-east-1a", "us-east-1b", "us-east-1c"]
}

variable "public_subnets" {
  type    = list(string)
  default = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
}

variable "private_subnets" {
  type    = list(string)
  default = ["10.0.10.0/24", "10.0.20.0/24", "10.0.30.0/24"]
}

# ── EKS ─────────────────────────────────────────────────────────────────────

variable "eks_version" {
  type    = string
  default = "1.29"
}

variable "eks_node_groups" {
  type = map(object({
    instance_type = string
    desired_size  = number
    max_size      = number
    min_size      = number
    capacity_type = optional(string, "ON_DEMAND")
  }))
  default = {
    default = {
      instance_type = "t3.medium"
      desired_size  = 3
      max_size      = 10
      min_size      = 2
    }
  }
}

# ── RDS ─────────────────────────────────────────────────────────────────────

variable "pg_version" {
  type    = string
  default = "16"
}

variable "db_username" {
  type    = string
  default = "geocore"
}

variable "db_password" {
  type      = string
  sensitive = true
}

variable "rds_instance_class" {
  type    = string
  default = "db.t4g.medium"
}

variable "rds_storage" {
  type    = number
  default = 100
}

variable "rds_multi_az" {
  type    = bool
  default = true
}

# ── Redis ────────────────────────────────────────────────────────────────────

variable "redis_node_type" {
  type    = string
  default = "cache.t3.micro"
}

variable "redis_nodes" {
  type    = number
  default = 2
}

variable "redis_multi_az" {
  type    = bool
  default = false
}

variable "redis_auth_token" {
  type      = string
  sensitive = true
  default   = ""
}

# ── Kafka ────────────────────────────────────────────────────────────────────

variable "kafka_version" {
  type    = string
  default = "3.5.1"
}

variable "kafka_brokers" {
  type    = number
  default = 3
}

variable "kafka_instance_type" {
  type    = string
  default = "kafka.m5.large"
}

variable "kafka_storage" {
  type    = number
  default = 100
}

# ── ALB ─────────────────────────────────────────────────────────────────────

variable "certificate_arn" {
  type = string
}

variable "alb_log_bucket" {
  type    = string
  default = ""
}

# ── Security ────────────────────────────────────────────────────────────────

variable "kms_key_arn" {
  type    = string
  default = ""
}
