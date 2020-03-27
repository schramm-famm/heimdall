output "internal_lb_dns_name" {
  value = aws_lb.heimdall-internal.dns_name
}

output "external_lb_dns_name" {
  value = aws_lb.heimdall-external.dns_name
}
