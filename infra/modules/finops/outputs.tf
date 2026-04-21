output "s3_endpoint_id" {
  value = aws_vpc_endpoint.s3.id
}

output "archive_bucket" {
  value = aws_s3_bucket.archive.bucket
}

output "budget_name" {
  value = aws_budgets_budget.monthly.name
}

output "app_log_group" {
  value = aws_cloudwatch_log_group.app.name
}
