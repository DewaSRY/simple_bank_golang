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
  description = "Port the core-service container listens on. Only reachable from the instance itself (127.0.0.1) and from nginx over the internal docker network — not exposed to the internet directly."
  type        = number
  default     = 8080
}

variable "docker_compose_version" {
  description = "docker compose CLI plugin release to install on the instance at boot (see https://github.com/docker/compose/releases). Defaults to a pinned version for reproducibility; set to \"latest\" to resolve GitHub's current release at boot time instead (retried and validated, unlike the old unconditional latest-lookup)."
  type        = string
  default     = "v5.5.1"
}

variable "ssh_cidr_blocks" {
  description = "CIDR blocks allowed to SSH into the instance"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "app_cidr_blocks" {
  description = "CIDR blocks allowed to reach nginx_port, i.e. the public entrypoint that reverse-proxies to core-service"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

# --- nginx: reverse proxy + rate limiter in front of core-service ---

variable "nginx_image" {
  description = "nginx Docker Hub image to run as the reverse proxy in front of core-service"
  type        = string
  default     = "nginx:1.27-alpine"
}

variable "nginx_port" {
  description = "Public port nginx listens on and proxies to core-service on app_port"
  type        = number
  default     = 80
}

variable "nginx_rate_limit_rps" {
  description = "nginx limit_req rate, in requests/second per client IP, enforced at the edge before a request reaches core-service"
  type        = number
  default     = 10
}

variable "nginx_rate_limit_burst" {
  description = "nginx limit_req burst size: how many requests over nginx_rate_limit_rps a client can burst before nginx starts responding 429"
  type        = number
  default     = 20
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

variable "app_rate_limit_enabled" {
  description = "Passed through as RATE_LIMIT_ENABLED — the in-process per-IP token-bucket limiter inside core-service itself, kept on behind nginx's edge limiter for defense in depth"
  type        = bool
  default     = true
}

variable "app_rate_limit_rps" {
  description = "Passed through as RATE_LIMIT_REQUESTS_PER_SECOND (core-service's own limiter). Set below nginx_rate_limit_rps since nginx already sheds the worst of it at the edge."
  type        = number
  default     = 5
}

variable "app_rate_limit_burst" {
  description = "Passed through as RATE_LIMIT_BURST (core-service's own limiter)"
  type        = number
  default     = 20
}

variable "log_level" {
  description = "Log level for the application, passed through as LOG_LEVEL (e.g. debug, info, warn, error)"
  type        = string
  default     = "info"
}

variable "log_format" {
  description = "Log format for the application, passed through as LOG_FORMAT (e.g. json, text)"
  type        = string
  default     = "json"
}
