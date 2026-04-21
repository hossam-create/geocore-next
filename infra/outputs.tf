# ── Root Outputs ────────────────────────────────────────────────────────────

output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "rds_endpoint" {
  description = "RDS PostgreSQL endpoint"
  value       = module.rds.endpoint
}

output "redis_endpoint" {
  description = "ElastiCache Redis primary endpoint"
  value       = module.redis.endpoint
}

output "kafka_bootstrap" {
  description = "MSK Kafka bootstrap brokers"
  value       = module.kafka.bootstrap_brokers
}

output "alb_dns" {
  description = "ALB DNS name (CNAME target)"
  value       = module.alb.alb_dns_name
}

output "target_group_arn" {
  description = "ALB target group ARN for Kubernetes Ingress"
  value       = module.alb.target_group_arn
}
