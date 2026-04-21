variable "db_password" {
  description = "RDS master password (from SSM/Vault)"
  type        = string
  sensitive   = true
}

variable "certificate_arn" {
  description = "ACM certificate ARN for HTTPS"
  type        = string
}

variable "alert_emails" {
  description = "Email addresses for AWS Budget alerts"
  type        = list(string)
  default     = []
}
