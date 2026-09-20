output "instance_id" {
  description = "EC2 instance ID"
  value       = aws_instance.core_service.id
}

output "public_ip" {
  description = "Public IP of the core-service instance"
  value       = aws_instance.core_service.public_ip
}

output "app_url" {
  description = "URL to reach the core-service API, through nginx"
  value       = "http://${aws_instance.core_service.public_ip}:${var.nginx_port}"
}

output "app_swagger_url" {
  description = "URL to reach the core-service Swagger API, through nginx"
  value       = "http://${aws_instance.core_service.public_ip}:${var.nginx_port}/swagger/index.html#/"
}

output "ssh_command" {
  description = "Command to SSH into the instance"
  value       = "ssh -i ${local_file.private_key.filename} ec2-user@${aws_instance.core_service.public_ip}"
}

output "rendered_user_data" {
  description = "The exact first-boot script aws_instance.core_service was given. EC2 only runs it once, on first boot, so `make tf-redeploy` pipes this over SSH to bring an already-running instance in line with the current config (new image, new nginx/rate-limit settings, etc). Sensitive: embeds db_source and jwt_secret_key in plaintext."
  value       = local.user_data
  sensitive   = true
}
