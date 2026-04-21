output "vpc_id" {
  value = aws_vpc.this.id
}

output "vpc_cidr" {
  value = aws_vpc.this.cidr_block
}

output "public_subnet_ids" {
  value = aws_subnet.public[*].id
}

output "private_subnet_ids" {
  value = aws_subnet.private[*].id
}

output "default_sg_id" {
  value = aws_security_group.default.id
}

output "db_subnet_group" {
  value = aws_db_subnet_group.this.name
}

output "cache_subnet_group" {
  value = aws_elasticache_subnet_group.this.name
}

output "nat_gateway_id" {
  value = aws_nat_gateway.this.id
}

output "private_route_table_id" {
  value = aws_route_table.private.id
}
