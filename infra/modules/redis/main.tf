# ── ElastiCache Redis Module ────────────────────────────────────────────────
# Creates: Redis replication group (cluster mode disabled), SG

resource "aws_security_group" "redis" {
  name        = "${var.cluster_id}-redis-sg"
  description = "ElastiCache Redis security group"
  vpc_id      = var.vpc_id

  ingress {
    description     = "Redis from VPC"
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    cidr_blocks     = [var.vpc_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(var.tags, {
    Name = "${var.cluster_id}-redis-sg"
  })
}

resource "aws_elasticache_replication_group" "this" {
  replication_group_id          = var.cluster_id
  replication_group_description = "GeoCore Redis cluster"

  engine         = "redis"
  engine_version = var.engine_version
  node_type      = var.node_type

  number_cache_clusters = var.num_cache_nodes
  parameter_group_name  = var.parameter_group_name

  subnet_group_name  = var.subnet_group_name
  security_group_ids = [aws_security_group.redis.id]

  at_rest_encryption_enabled    = true
  transit_encryption_enabled    = true
  auth_token                    = var.auth_token != "" ? var.auth_token : null

  automatic_failover_enabled = var.num_cache_nodes >= 2
  multi_az_enabled           = var.multi_az

  snapshot_retention_limit = var.snapshot_retention_days
  snapshot_window          = "03:00-05:00"

  maintenance_window = "Mon:05:00-Mon:06:00"

  tags = merge(var.tags, {
    Name = var.cluster_id
  })
}
