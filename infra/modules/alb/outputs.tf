output "alb_arn" {
  value = aws_lb.this.arn
}

output "alb_dns_name" {
  description = "ALB DNS name (use as CNAME target)"
  value       = aws_lb.this.dns_name
}

output "alb_zone_id" {
  value = aws_lb.this.zone_id
}

output "target_group_arn" {
  description = "API target group ARN (for EKS Ingress annotations)"
  value       = aws_lb_target_group.api.arn
}

output "security_group_id" {
  value = aws_security_group.alb.id
}
