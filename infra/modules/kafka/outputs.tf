output "bootstrap_brokers" {
  description = "Kafka bootstrap brokers (plaintext)"
  value       = aws_msk_cluster.this.bootstrap_brokers
}

output "bootstrap_brokers_tls" {
  description = "Kafka bootstrap brokers (TLS)"
  value       = aws_msk_cluster.this.bootstrap_brokers_tls
}

output "cluster_arn" {
  value = aws_msk_cluster.this.arn
}

output "security_group_id" {
  value = aws_security_group.kafka.id
}

output "zookeeper_connect" {
  value = aws_msk_cluster.this.zookeeper_connect_string
}
