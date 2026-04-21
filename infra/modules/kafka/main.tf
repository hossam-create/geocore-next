# ── MSK Managed Kafka Module ────────────────────────────────────────────────
# Creates: MSK cluster, SG, configuration (9 aggregate topics + DLQs)

resource "aws_security_group" "kafka" {
  name        = "${var.cluster_name}-kafka-sg"
  description = "MSK Kafka security group"
  vpc_id      = var.vpc_id

  ingress {
    description     = "Kafka brokers from VPC"
    from_port       = 9092
    to_port         = 9092
    protocol        = "tcp"
    cidr_blocks     = [var.vpc_cidr]
  }

  ingress {
    description     = "Kafka TLS from VPC"
    from_port       = 9094
    to_port         = 9094
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
    Name = "${var.cluster_name}-kafka-sg"
  })
}

# ── MSK Configuration (custom settings) ─────────────────────────────────────

resource "aws_msk_configuration" "this" {
  name           = "${var.cluster_name}-config"
  kafka_versions = [var.kafka_version]

  server_properties = <<-PROPERTIES
    auto.create.topics.enable=false
    default.replication.factor=${var.number_of_broker_nodes}
    min.insync.replicas=2
    num.partitions=6
    log.retention.hours=168
    message.max.bytes=10485760
  PROPERTIES
}

# ── MSK Cluster ─────────────────────────────────────────────────────────────

resource "aws_msk_cluster" "this" {
  cluster_name           = var.cluster_name
  kafka_version          = var.kafka_version
  number_of_broker_nodes = var.number_of_broker_nodes

  broker_node_group_info {
    instance_type   = var.instance_type
    client_subnets  = var.private_subnet_ids
    security_groups = [aws_security_group.kafka.id]

    storage_info {
      ebs_storage {
        volume_size = var.storage_size_gb
      }
    }
  }

  configuration_info {
    arn      = aws_msk_configuration.this.arn
    revision = aws_msk_configuration.this.latest_revision
  }

  encryption_info {
    encryption_at_rest_kms_key_arn = var.kms_key_arn != "" ? var.kms_key_arn : null
    encryption_in_cluster {
      data_volume_kms_key = var.kms_key_arn != "" ? var.kms_key_arn : null
    }
  }

  open_monitoring {
    prometheus {
      jmx_exporter {
        enabled_in_broker = true
      }
      node_exporter {
        enabled_in_broker = true
      }
    }
  }

  logging_info {
    broker_logs {
      cloudwatch_logs {
        enabled   = true
        log_group = aws_cloudwatch_log_group.kafka.name
      }
      firehose {
        enabled = false
      }
      s3 {
        enabled = false
      }
    }
  }

  tags = merge(var.tags, {
    Name = var.cluster_name
  })
}

resource "aws_cloudwatch_log_group" "kafka" {
  name              = "/aws/msk/${var.cluster_name}"
  retention_in_days = 14

  tags = var.tags
}
