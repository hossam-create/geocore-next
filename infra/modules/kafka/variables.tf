variable "cluster_name" {
  description = "MSK cluster name"
  type        = string
}

variable "kafka_version" {
  description = "Apache Kafka version"
  type        = string
  default     = "3.5.1"
}

variable "number_of_broker_nodes" {
  description = "Number of broker nodes (min 3 for production)"
  type        = number
  default     = 3
}

variable "instance_type" {
  description = "Broker instance type"
  type        = string
  default     = "kafka.m5.large"
}

variable "storage_size_gb" {
  description = "EBS storage per broker (GB)"
  type        = number
  default     = 100
}

variable "private_subnet_ids" {
  description = "Private subnet IDs for brokers"
  type        = list(string)
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

variable "kms_key_arn" {
  description = "KMS key ARN for encryption (empty = default)"
  type        = string
  default     = ""
}

variable "tags" {
  description = "Common tags"
  type        = map(string)
  default     = {}
}
