# ── PRODUCTION Environment ──────────────────────────────────────────────────
# Full HA: multi-AZ everything, larger instances, deletion protection, encryption

terraform {
  backend "s3" {
    bucket = "geocore-tfstate-prod"
    key    = "terraform.tfstate"
    region = "us-east-1"
  }
}

provider "aws" {
  region = "us-east-1"

  default_tags {
    tags = {
      Environment = "prod"
      Project     = "geocore"
      ManagedBy   = "terraform"
      CostCenter  = "engineering"
    }
  }
}

module "vpc" {
  source = "../../modules/vpc"

  name = "geocore-prod"
  cidr = "10.0.0.0/16"

  azs             = ["us-east-1a", "us-east-1b", "us-east-1c"]
  public_subnets  = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  private_subnets = ["10.0.10.0/24", "10.0.20.0/24", "10.0.30.0/24"]

  tags = { Environment = "prod" }
}

module "eks" {
  source = "../../modules/eks"

  cluster_name    = "geocore-prod"
  cluster_version = "1.29"
  vpc_id          = module.vpc.vpc_id
  vpc_cidr        = module.vpc.vpc_cidr
  private_subnet_ids = module.vpc.private_subnet_ids
  kms_key_arn     = var.kms_key_arn

  node_groups = {
    default = {
      instance_type = "t3.medium"
      desired_size  = 3
      max_size      = 10
      min_size      = 2
    }
    critical = {
      instance_type = "t3.large"
      desired_size  = 2
      max_size      = 6
      min_size      = 2
    }
  }

  public_access_cidrs = ["10.0.0.0/16"] # No public API access

  tags = { Environment = "prod" }
}

module "rds" {
  source = "../../modules/rds"

  identifier        = "geocore-prod"
  instance_class    = "db.r6g.large"
  allocated_storage = 200
  multi_az          = true
  username          = "geocore"
  password          = var.db_password
  vpc_id            = module.vpc.vpc_id
  vpc_cidr          = module.vpc.vpc_cidr
  subnet_group_name = module.vpc.db_subnet_group

  backup_retention_days = 30
  deletion_protection   = true
  skip_final_snapshot   = false
  performance_insights  = true

  tags = { Environment = "prod" }
}

module "redis" {
  source = "../../modules/redis"

  cluster_id        = "geocore-prod"
  node_type         = "cache.r6g.large"
  num_cache_nodes   = 3
  subnet_group_name = module.vpc.cache_subnet_group
  vpc_id            = module.vpc.vpc_id
  vpc_cidr          = module.vpc.vpc_cidr
  auth_token        = var.redis_auth_token
  multi_az          = true
  snapshot_retention_days = 7

  tags = { Environment = "prod" }
}

module "kafka" {
  source = "../../modules/kafka"

  cluster_name        = "geocore-prod"
  number_of_broker_nodes = 3
  instance_type       = "kafka.m5.xlarge"
  storage_size_gb     = 500
  private_subnet_ids  = module.vpc.private_subnet_ids
  vpc_id              = module.vpc.vpc_id
  vpc_cidr            = module.vpc.vpc_cidr
  kms_key_arn         = var.kms_key_arn

  tags = { Environment = "prod" }
}

module "alb" {
  source = "../../modules/alb"

  name              = "geocore-prod"
  vpc_id            = module.vpc.vpc_id
  public_subnet_ids = module.vpc.public_subnet_ids
  certificate_arn   = var.certificate_arn

  enable_deletion_protection = true
  log_bucket                 = var.log_bucket

  tags = { Environment = "prod" }
}

# ── FinOps (cost optimization) ─────────────────────────────────────────────
# Prod: NO scheduled stop, higher budget, longer retention, all VPC endpoints

module "finops" {
  source = "../../modules/finops"

  project     = "geocore"
  environment = "prod"
  vpc_id      = module.vpc.vpc_id
  private_subnet_ids      = module.vpc.private_subnet_ids
  private_route_table_ids = [module.vpc.private_route_table_id]
  default_sg_id           = module.vpc.default_sg_id

  monthly_budget    = 500
  alert_emails      = var.alert_emails
  log_retention_days = 14

  enable_scheduled_stop = false  # NEVER stop prod
  rds_identifier        = ""

  tags = { Environment = "prod" }
}
