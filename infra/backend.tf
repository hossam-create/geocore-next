# ── Remote State ────────────────────────────────────────────────────────────
# Uncomment and configure for your AWS account.
# Each environment overrides this in envs/{dev,staging,prod}/main.tf

# terraform {
#   backend "s3" {
#     bucket         = "geocore-tfstate"
#     key            = "infra/terraform.tfstate"
#     region         = "us-east-1"
#     dynamodb_table = "geocore-tfstate-lock"
#     encrypt        = true
#   }
# }
