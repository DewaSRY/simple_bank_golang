variable "aws_region" {
  description = "AWS region to deploy into"
  type        = string
  default     = "ap-southeast-1"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}

variable "docker_image" {
  description = "Docker Hub image to run on the instance, e.g. sdewa/core-service-dep:latest"
  type        = string
  default     = "sdewa/core-service-dep:latest"
}

variable "app_port" {
  description = "Port the core-service container listens on and exposes"
  type        = number
  default     = 8080
}

variable "ssh_cidr_blocks" {
  description = "CIDR blocks allowed to SSH into the instance"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "app_cidr_blocks" {
  description = "CIDR blocks allowed to reach the app port"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

# --- App config (mirrors internal/config.Config / app.env) ---

variable "db_driver" {
  description = "SQL driver name, passed through as DB_DRIVER"
  type        = string
  default     = "postgres"
}

variable "db_source" {
  description = "Supabase Postgres connection string, passed through as DB_SOURCE (e.g. postgresql://user:pass@host:5432/postgres?sslmode=require)"
  type        = string
  sensitive   = true
}

variable "jwt_secret_key" {
  description = "JWT signing secret, passed through as JWT_SECRET_KEY (must be at least 32 characters)"
  type        = string
  sensitive   = true

  validation {
    condition     = length(var.jwt_secret_key) >= 32
    error_message = "jwt_secret_key must be at least 32 characters."
  }
}

variable "jwt_access_token_duration" {
  description = "Access token TTL, passed through as JWT_ACCESS_TOKEN_DURATION"
  type        = string
  default     = "60m"
}

variable "cors_allowed_origins" {
  description = "Comma-separated list of allowed CORS origins, passed through as CORS_ALLOWED_ORIGINS (blank disables CORS)"
  type        = string
  default     = ""
}
