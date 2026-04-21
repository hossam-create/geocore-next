variable "cluster_name" {
  description = "EKS cluster name"
  type        = string
}

variable "cluster_version" {
  description = "Kubernetes version"
  type        = string
  default     = "1.29"
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "vpc_cidr" {
  description = "VPC CIDR for SG rules"
  type        = string
  default     = "10.0.0.0/16"
}

variable "private_subnet_ids" {
  description = "Private subnet IDs for EKS nodes"
  type        = list(string)
}

variable "node_groups" {
  description = "Node group definitions"
  type = map(object({
    instance_type = string
    desired_size  = number
    max_size      = number
    min_size      = number
    capacity_type = optional(string, "ON_DEMAND")
  }))
  default = {
    default = {
      instance_type = "t3.medium"
      desired_size  = 3
      max_size      = 10
      min_size      = 2
    }
  }
}

variable "public_access_cidrs" {
  description = "CIDRs allowed for public API endpoint"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "kms_key_arn" {
  description = "KMS key ARN for secret encryption (empty = default)"
  type        = string
  default     = ""
}

variable "tags" {
  description = "Common tags"
  type        = map(string)
  default     = {}
}
