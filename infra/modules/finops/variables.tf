variable "project" {
  type = string
}

variable "environment" {
  type = string
}

variable "region" {
  type    = string
  default = "us-east-1"
}

variable "vpc_id" {
  type = string
}

variable "private_subnet_ids" {
  type = list(string)
}

variable "private_route_table_ids" {
  type = list(string)
}

variable "default_sg_id" {
  type = string
}

# ── Budget ──────────────────────────────────────────────────────────────────

variable "monthly_budget" {
  description = "Monthly AWS budget in USD"
  type        = number
  default     = 100
}

variable "alert_emails" {
  description = "Email addresses for budget alerts"
  type        = list(string)
  default     = []
}

# ── Log Retention ───────────────────────────────────────────────────────────

variable "log_retention_days" {
  description = "CloudWatch log retention days"
  type        = number
  default     = 7
}

# ── Scheduled Stop (staging/dev) ────────────────────────────────────────────

variable "enable_scheduled_stop" {
  description = "Enable RDS auto-stop at night (dev/staging only)"
  type        = bool
  default     = false
}

variable "rds_identifier" {
  description = "RDS instance identifier for scheduled stop"
  type        = string
  default     = ""
}

# ── Tags ────────────────────────────────────────────────────────────────────

variable "tags" {
  type    = map(string)
  default = {}
}
