# ── GeoCore Infrastructure Root Module ─────────────────────────────────────
# Wire all modules together. Use environment-specific configs in envs/ instead.

module "vpc" {
  source = "./modules/vpc"

  name = "${var.project}-${var.environment}"
  cidr = var.vpc_cidr

  azs             = var.azs
  public_subnets  = var.public_subnets
  private_subnets = var.private_subnets

  tags = local.common_tags
}

module "eks" {
  source = "./modules/eks"

  cluster_name    = "${var.project}-${var.environment}"
  cluster_version = var.eks_version
  vpc_id          = module.vpc.vpc_id
  vpc_cidr        = module.vpc.vpc_cidr
  private_subnet_ids = module.vpc.private_subnet_ids
  kms_key_arn     = var.kms_key_arn

  node_groups = var.eks_node_groups

  tags = local.common_tags
}

module "rds" {
  source = "./modules/rds"

  identifier        = "${var.project}-${var.environment}"
  engine_version    = var.pg_version
  instance_class    = var.rds_instance_class
  allocated_storage = var.rds_storage
  multi_az          = var.rds_multi_az
  username          = var.db_username
  password          = var.db_password
  vpc_id            = module.vpc.vpc_id
  vpc_cidr          = module.vpc.vpc_cidr
  subnet_group_name = module.vpc.db_subnet_group

  deletion_protection  = var.environment == "prod"
  skip_final_snapshot  = var.environment != "prod"
  performance_insights = var.environment != "dev"

  tags = local.common_tags
}

module "redis" {
  source = "./modules/redis"

  cluster_id        = "${var.project}-${var.environment}"
  node_type         = var.redis_node_type
  num_cache_nodes   = var.redis_nodes
  subnet_group_name = module.vpc.cache_subnet_group
  vpc_id            = module.vpc.vpc_id
  vpc_cidr          = module.vpc.vpc_cidr
  auth_token        = var.redis_auth_token
  multi_az          = var.redis_multi_az

  tags = local.common_tags
}

module "kafka" {
  source = "./modules/kafka"

  cluster_name        = "${var.project}-${var.environment}"
  kafka_version        = var.kafka_version
  number_of_broker_nodes = var.kafka_brokers
  instance_type       = var.kafka_instance_type
  storage_size_gb     = var.kafka_storage
  private_subnet_ids  = module.vpc.private_subnet_ids
  vpc_id              = module.vpc.vpc_id
  vpc_cidr            = module.vpc.vpc_cidr
  kms_key_arn         = var.kms_key_arn

  tags = local.common_tags
}

module "alb" {
  source = "./modules/alb"

  name              = "${var.project}-${var.environment}"
  vpc_id            = module.vpc.vpc_id
  public_subnet_ids = module.vpc.public_subnet_ids
  certificate_arn   = var.certificate_arn

  enable_deletion_protection = var.environment == "prod"
  log_bucket                 = var.alb_log_bucket

  tags = local.common_tags
}

# ── Locals ──────────────────────────────────────────────────────────────────

locals {
  common_tags = {
    Project     = var.project
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}
