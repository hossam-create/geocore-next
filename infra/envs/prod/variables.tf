variable "db_password" {
  description = "RDS master password (from SSM Parameter Store)"
  type        = string
  sensitive   = true
}

variable "certificate_arn" {
  description = "ACM certificate ARN for HTTPS"
  type        = string
}

variable "kms_key_arn" {
  description = "KMS key ARN for EKS + MSK encryption"
  type        = string
  default     = ""
}

variable "redis_auth_token" {
  description = "Redis AUTH token"
  type        = string
  sensitive   = true
}

variable "log_bucket" {
  description = "S3 bucket for ALB access logs"
  type        = string
  default     = ""
}

variable "alert_emails" {
  description = "Email addresses for AWS Budget alerts"
  type        = list(string)
  default     = []
}
