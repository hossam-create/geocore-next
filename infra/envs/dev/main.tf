# ── DEV Environment ─────────────────────────────────────────────────────────
# Cost-optimized: single NAT, smaller instances, no multi-AZ RDS

terraform {
  backend "s3" {
    bucket = "geocore-tfstate-dev"
    key    = "terraform.tfstate"
    region = "us-east-1"
  }
}

provider "aws" {
  region = "us-east-1"

  default_tags {
    tags = {
      Environment = "dev"
      Project     = "geocore"
      ManagedBy   = "terraform"
    }
  }
}

module "vpc" {
  source = "../../modules/vpc"

  name = "geocore-dev"
  cidr = "10.0.0.0/16"

  azs             = ["us-east-1a", "us-east-1b"]
  public_subnets  = ["10.0.1.0/24", "10.0.2.0/24"]
  private_subnets = ["10.0.10.0/24", "10.0.20.0/24"]

  tags = { Environment = "dev" }
}

module "eks" {
  source = "../../modules/eks"

  cluster_name    = "geocore-dev"
  cluster_version = "1.29"
  vpc_id          = module.vpc.vpc_id
  vpc_cidr        = module.vpc.vpc_cidr
  private_subnet_ids = module.vpc.private_subnet_ids

  node_groups = {
    default = {
      instance_type = "t3.small"
      desired_size  = 2
      max_size      = 4
      min_size      = 1
      capacity_type = "SPOT"
    }
  }

  tags = { Environment = "dev" }
}

module "rds" {
  source = "../../modules/rds"

  identifier        = "geocore-dev"
  instance_class    = "db.t4g.small"
  allocated_storage = 50
  multi_az          = false
  username          = "geocore"
  password          = var.db_password
  vpc_id            = module.vpc.vpc_id
  vpc_cidr          = module.vpc.vpc_cidr
  subnet_group_name = module.vpc.db_subnet_group

  deletion_protection  = false
  skip_final_snapshot  = true
  performance_insights = false

  tags = { Environment = "dev" }
}

module "redis" {
  source = "../../modules/redis"

  cluster_id        = "geocore-dev"
  node_type         = "cache.t3.micro"
  num_cache_nodes   = 1
  subnet_group_name = module.vpc.cache_subnet_group
  vpc_id            = module.vpc.vpc_id
  vpc_cidr          = module.vpc.vpc_cidr
  multi_az          = false

  tags = { Environment = "dev" }
}

module "kafka" {
  source = "../../modules/kafka"

  cluster_name        = "geocore-dev"
  number_of_broker_nodes = 2
  instance_type       = "kafka.t3.small"
  storage_size_gb     = 50
  private_subnet_ids  = module.vpc.private_subnet_ids
  vpc_id              = module.vpc.vpc_id
  vpc_cidr            = module.vpc.vpc_cidr

  tags = { Environment = "dev" }
}

module "alb" {
  source = "../../modules/alb"

  name              = "geocore-dev"
  vpc_id            = module.vpc.vpc_id
  public_subnet_ids = module.vpc.public_subnet_ids
  certificate_arn   = var.certificate_arn

  enable_deletion_protection = false

  tags = { Environment = "dev" }
}

# ── FinOps (cost optimization) ─────────────────────────────────────────────

module "finops" {
  source = "../../modules/finops"

  project     = "geocore"
  environment = "dev"
  vpc_id      = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  private_route_table_ids = [module.vpc.private_route_table_id]
  default_sg_id         = module.vpc.default_sg_id

  monthly_budget    = 50
  alert_emails      = var.alert_emails
  log_retention_days = 3

  enable_scheduled_stop = true
  rds_identifier        = "geocore-dev"

  tags = { Environment = "dev" }
}
