output "eks_cluster_name" {
  value = module.eks.cluster_name
}

output "rds_endpoint" {
  value = module.rds.endpoint
}

output "redis_endpoint" {
  value = module.redis.endpoint
}

output "kafka_bootstrap" {
  value = module.kafka.bootstrap_brokers
}

output "alb_dns" {
  value = module.alb.alb_dns_name
}
