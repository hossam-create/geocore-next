# ── STAGING Environment ─────────────────────────────────────────────────────
# Production-like but cost-aware: multi-AZ RDS, 2 Redis nodes, 3 Kafka brokers

terraform {
  backend "s3" {
    bucket = "geocore-tfstate-staging"
    key    = "terraform.tfstate"
    region = "us-east-1"
  }
}

provider "aws" {
  region = "us-east-1"

  default_tags {
    tags = {
      Environment = "staging"
      Project     = "geocore"
      ManagedBy   = "terraform"
    }
  }
}

module "vpc" {
  source = "../../modules/vpc"

  name = "geocore-staging"
  cidr = "10.1.0.0/16"

  azs             = ["us-east-1a", "us-east-1b", "us-east-1c"]
  public_subnets  = ["10.1.1.0/24", "10.1.2.0/24", "10.1.3.0/24"]
  private_subnets = ["10.1.10.0/24", "10.1.20.0/24", "10.1.30.0/24"]

  tags = { Environment = "staging" }
}

module "eks" {
  source = "../../modules/eks"

  cluster_name    = "geocore-staging"
  cluster_version = "1.29"
  vpc_id          = module.vpc.vpc_id
  vpc_cidr        = module.vpc.vpc_cidr
  private_subnet_ids = module.vpc.private_subnet_ids

  node_groups = {
    default = {
      instance_type = "t3.medium"
      desired_size  = 3
      max_size      = 6
      min_size      = 2
    }
  }

  tags = { Environment = "staging" }
}

module "rds" {
  source = "../../modules/rds"

  identifier        = "geocore-staging"
  instance_class    = "db.t4g.medium"
  allocated_storage = 100
  multi_az          = true
  username          = "geocore"
  password          = var.db_password
  vpc_id            = module.vpc.vpc_id
  vpc_cidr          = module.vpc.vpc_cidr
  subnet_group_name = module.vpc.db_subnet_group

  deletion_protection  = true
  skip_final_snapshot  = false
  performance_insights = true

  tags = { Environment = "staging" }
}

module "redis" {
  source = "../../modules/redis"

  cluster_id        = "geocore-staging"
  node_type         = "cache.t3.small"
  num_cache_nodes   = 2
  subnet_group_name = module.vpc.cache_subnet_group
  vpc_id            = module.vpc.vpc_id
  vpc_cidr          = module.vpc.vpc_cidr
  multi_az          = true

  tags = { Environment = "staging" }
}

module "kafka" {
  source = "../../modules/kafka"

  cluster_name        = "geocore-staging"
  number_of_broker_nodes = 3
  instance_type       = "kafka.m5.large"
  storage_size_gb     = 100
  private_subnet_ids  = module.vpc.private_subnet_ids
  vpc_id              = module.vpc.vpc_id
  vpc_cidr            = module.vpc.vpc_cidr

  tags = { Environment = "staging" }
}

module "alb" {
  source = "../../modules/alb"

  name              = "geocore-staging"
  vpc_id            = module.vpc.vpc_id
  public_subnet_ids = module.vpc.public_subnet_ids
  certificate_arn   = var.certificate_arn

  enable_deletion_protection = true

  tags = { Environment = "staging" }
}

# ── FinOps (cost optimization) ─────────────────────────────────────────────

module "finops" {
  source = "../../modules/finops"

  project     = "geocore"
  environment = "staging"
  vpc_id      = module.vpc.vpc_id
  private_subnet_ids      = module.vpc.private_subnet_ids
  private_route_table_ids = [module.vpc.private_route_table_id]
  default_sg_id           = module.vpc.default_sg_id

  monthly_budget    = 200
  alert_emails      = var.alert_emails
  log_retention_days = 7

  enable_scheduled_stop = true
  rds_identifier        = "geocore-staging"

  tags = { Environment = "staging" }
}
