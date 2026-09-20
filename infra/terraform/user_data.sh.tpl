#!/bin/bash
set -euxo pipefail

dnf update -y
dnf install -y docker
systemctl enable --now docker
usermod -aG docker ec2-user

# docker compose plugin — Amazon Linux 2023's `docker` package doesn't bundle
# it. Fetches whatever is currently latest rather than a pinned version,
# since this only runs at instance boot / redeploy time, not in CI.
mkdir -p /usr/local/lib/docker/cli-plugins
COMPOSE_VERSION=$(curl -fsSL https://api.github.com/repos/docker/compose/releases/latest | grep -m1 '"tag_name"' | cut -d '"' -f4)
curl -fsSL "https://github.com/docker/compose/releases/download/$COMPOSE_VERSION/docker-compose-linux-x86_64" -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

mkdir -p /opt/core-service

cat > /opt/core-service/app.env <<EOF
DB_DRIVER=${db_driver}
DB_SOURCE=${db_source}
SERVER_ADDRESS=0.0.0.0:${app_port}
JWT_SECRET_KEY=${jwt_secret_key}
JWT_ACCESS_TOKEN_DURATION=${jwt_access_token_duration}
CORS_ALLOWED_ORIGINS=${cors_allowed_origins}
RATE_LIMIT_ENABLED=${rate_limit_enabled}
RATE_LIMIT_REQUESTS_PER_SECOND=${rate_limit_rps}
RATE_LIMIT_BURST=${rate_limit_burst}
EOF
chmod 600 /opt/core-service/app.env

echo '${nginx_conf_base64}' | base64 -d > /opt/core-service/nginx.conf
chmod 644 /opt/core-service/nginx.conf

echo '${docker_compose_yml_base64}' | base64 -d > /opt/core-service/docker-compose.yml
chmod 644 /opt/core-service/docker-compose.yml

# Idempotent: pull + up -d each time, safe to rerun over SSH on an
# already-running instance (see docs/TERRAFORM_EC2_DEPLOY.md Section 7).
cd /opt/core-service
docker compose pull
docker compose up -d
