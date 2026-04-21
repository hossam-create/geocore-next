# ── FinOps Module ───────────────────────────────────────────────────────────
# Cost optimization: AWS Budgets, VPC endpoints (NAT savings), log retention,
# S3 lifecycle rules, RDS scheduled stop (staging/dev)

# ── AWS Budgets ─────────────────────────────────────────────────────────────

resource "aws_budgets_budget" "monthly" {
  name         = "${var.project}-${var.environment}-monthly"
  budget_type  = "COST"
  limit_amount = var.monthly_budget
  limit_unit   = "USD"
  time_unit    = "MONTHLY"

  cost_filter {
    name = "TagKeyValue"
    values = [
      "Project$${var.project}",
    ]
  }

  notification {
    comparison_operator       = "GREATER_THAN"
    threshold                 = 50
    threshold_type            = "PERCENTAGE"
    notification_type         = "ACTUAL"
    subscriber_email_addresses = var.alert_emails
  }

  notification {
    comparison_operator       = "GREATER_THAN"
    threshold                 = 80
    threshold_type            = "PERCENTAGE"
    notification_type         = "ACTUAL"
    subscriber_email_addresses = var.alert_emails
  }

  notification {
    comparison_operator       = "GREATER_THAN"
    threshold                 = 100
    threshold_type            = "PERCENTAGE"
    notification_type         = "ACTUAL"
    subscriber_email_addresses = var.alert_emails
  }

  # Forecasted overspend alert
  notification {
    comparison_operator       = "GREATER_THAN"
    threshold                 = 100
    threshold_type            = "PERCENTAGE"
    notification_type         = "FORECASTED"
    subscriber_email_addresses = var.alert_emails
  }

  tags = merge(var.tags, {
    Name = "${var.project}-${var.environment}-budget"
  })
}

# ── VPC Endpoints (NAT Gateway cost killer) ─────────────────────────────────
# All AWS service traffic stays inside AWS network — no NAT charges

resource "aws_vpc_endpoint" "s3" {
  vpc_id       = var.vpc_id
  service_name = "com.amazonaws.${var.region}.s3"
  vpc_endpoint_type = "Gateway"

  route_table_ids = var.private_route_table_ids

  tags = merge(var.tags, {
    Name = "${var.project}-${var.environment}-s3-endpoint"
  })
}

resource "aws_vpc_endpoint" "ecr_api" {
  vpc_id             = var.vpc_id
  service_name       = "com.amazonaws.${var.region}.ecr.api"
  vpc_endpoint_type  = "Interface"
  private_dns_enabled = true

  subnet_ids          = var.private_subnet_ids
  security_group_ids  = [var.default_sg_id]

  tags = merge(var.tags, {
    Name = "${var.project}-${var.environment}-ecr-api-endpoint"
  })
}

resource "aws_vpc_endpoint" "ecr_dkr" {
  vpc_id             = var.vpc_id
  service_name       = "com.amazonaws.${var.region}.ecr.dkr"
  vpc_endpoint_type  = "Interface"
  private_dns_enabled = true

  subnet_ids          = var.private_subnet_ids
  security_group_ids  = [var.default_sg_id]

  tags = merge(var.tags, {
    Name = "${var.project}-${var.environment}-ecr-dkr-endpoint"
  })
}

resource "aws_vpc_endpoint" "ssm" {
  vpc_id             = var.vpc_id
  service_name       = "com.amazonaws.${var.region}.ssm"
  vpc_endpoint_type  = "Interface"
  private_dns_enabled = true

  subnet_ids          = var.private_subnet_ids
  security_group_ids  = [var.default_sg_id]

  tags = merge(var.tags, {
    Name = "${var.project}-${var.environment}-ssm-endpoint"
  })
}

resource "aws_vpc_endpoint" "secretsmanager" {
  vpc_id             = var.vpc_id
  service_name       = "com.amazonaws.${var.region}.secretsmanager"
  vpc_endpoint_type  = "Interface"
  private_dns_enabled = true

  subnet_ids          = var.private_subnet_ids
  security_group_ids  = [var.default_sg_id]

  tags = merge(var.tags, {
    Name = "${var.project}-${var.environment}-secrets-endpoint"
  })
}

# ── CloudWatch Log Group Retention ──────────────────────────────────────────

resource "aws_cloudwatch_log_group" "app" {
  name              = "/aws/geocore/${var.environment}/app"
  retention_in_days = var.log_retention_days

  tags = var.tags
}

resource "aws_cloudwatch_log_group" "eks" {
  name              = "/aws/eks/${var.project}-${var.environment}"
  retention_in_days = var.log_retention_days

  tags = var.tags
}

# ── S3 Lifecycle — Archive cold data ────────────────────────────────────────

resource "aws_s3_bucket" "archive" {
  bucket = "${var.project}-${var.environment}-archive"

  tags = merge(var.tags, {
    Name = "${var.project}-${var.environment}-archive"
  })
}

resource "aws_s3_bucket_lifecycle_configuration" "archive" {
  bucket = aws_s3_bucket.archive.id

  rule {
    id     = "archive-old-data"
    status = "Enabled"

    transition {
      days          = 90
      storage_class = "STANDARD_IA"
    }

    transition {
      days          = 365
      storage_class = "GLACIER"
    }

    expiration {
      days = 2555 # 7 years
    }
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "archive" {
  bucket = aws_s3_bucket.archive.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# ── RDS Scheduled Stop (staging/dev only) ───────────────────────────────────
# Stops RDS at night to save ~60% of RDS cost

resource "aws_scheduler_schedule" "rds_stop" {
  count = var.enable_scheduled_stop ? 1 : 0

  name = "${var.project}-${var.environment}-rds-stop"

  flexible_time_window {
    mode = "OFF"
  }

  schedule_expression = "cron(0 0 ? * MON-FRI *)" # midnight UTC Mon-Fri

  target {
    arn      = "arn:aws:scheduler:::aws-sdk:rds:stopDBCluster"
    role_arn = aws_iam_role.scheduler[0].arn

    input = jsonencode({
      DBClusterIdentifier = var.rds_identifier
    })
  }
}

resource "aws_scheduler_schedule" "rds_start" {
  count = var.enable_scheduled_stop ? 1 : 0

  name = "${var.project}-${var.environment}-rds-start"

  flexible_time_window {
    mode = "OFF"
  }

  schedule_expression = "cron(0 8 ? * MON-FRI *)" # 8am UTC Mon-Fri

  target {
    arn      = "arn:aws:scheduler:::aws-sdk:rds:startDBCluster"
    role_arn = aws_iam_role.scheduler[0].arn

    input = jsonencode({
      DBClusterIdentifier = var.rds_identifier
    })
  }
}

resource "aws_iam_role" "scheduler" {
  count = var.enable_scheduled_stop ? 1 : 0

  name = "${var.project}-${var.environment}-scheduler-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "scheduler.amazonaws.com"
      }
    }]
  })

  tags = var.tags
}

resource "aws_iam_role_policy" "scheduler_rds" {
  count = var.enable_scheduled_stop ? 1 : 0

  name = "${var.project}-${var.environment}-scheduler-rds-policy"
  role = aws_iam_role.scheduler[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["rds:StopDBCluster", "rds:StartDBCluster"]
      Resource = "*"
    }]
  })
}
