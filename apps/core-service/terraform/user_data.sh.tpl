#!/bin/bash
set -euxo pipefail

dnf update -y
dnf install -y docker
systemctl enable --now docker
usermod -aG docker ec2-user

docker pull ${docker_image}

docker rm -f core-service || true

docker run -d \
  --name core-service \
  --restart unless-stopped \
  -p ${app_port}:${app_port} \
  -e DB_DRIVER="${db_driver}" \
  -e DB_SOURCE="${db_source}" \
  -e SERVER_ADDRESS="0.0.0.0:${app_port}" \
  -e JWT_SECRET_KEY="${jwt_secret_key}" \
  -e JWT_ACCESS_TOKEN_DURATION="${jwt_access_token_duration}" \
  -e CORS_ALLOWED_ORIGINS="${cors_allowed_origins}" \
  ${docker_image}
