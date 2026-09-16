#!/bin/bash
set -euxo pipefail

dnf update -y
dnf install -y docker
systemctl enable --now docker
usermod -aG docker ec2-user

docker pull ${docker_image}

docker rm -f core-service || true

mkdir -p /opt/core-service
cat > /opt/core-service/app.env <<EOF
DB_DRIVER=${db_driver}
DB_SOURCE=${db_source}
SERVER_ADDRESS=0.0.0.0:${app_port}
JWT_SECRET_KEY=${jwt_secret_key}
JWT_ACCESS_TOKEN_DURATION=${jwt_access_token_duration}
CORS_ALLOWED_ORIGINS=${cors_allowed_origins}
EOF
chmod 600 /opt/core-service/app.env

docker run -d \
  --name core-service \
  --restart unless-stopped \
  -p ${app_port}:${app_port} \
  -v /opt/core-service/app.env:/app/app.env:ro \
  ${docker_image}