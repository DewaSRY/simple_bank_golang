output "instance_id" {
  description = "EC2 instance ID"
  value       = aws_instance.core_service.id
}

output "public_ip" {
  description = "Public IP of the core-service instance"
  value       = aws_instance.core_service.public_ip
}

output "app_url" {
  description = "URL to reach the core-service API"
  value       = "http://${aws_instance.core_service.public_ip}:${var.app_port}"
}

output "app_swagger_url" {
  description = "URL to reach the core-service Swagger API"
  value       = "http://${aws_instance.core_service.public_ip}:${var.app_port}/swagger/index.html#/"
}

output "ssh_command" {
  description = "Command to SSH into the instance"
  value       = "ssh -i ${local_file.private_key.filename} ec2-user@${aws_instance.core_service.public_ip}"
}
