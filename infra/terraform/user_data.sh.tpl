#!/bin/bash
set -euxo pipefail

dnf update -y
dnf install -y docker
systemctl enable --now docker
usermod -aG docker ec2-user

# Shared network so nginx can reach core-service by container name; neither
# container needs to publish app_port on the host for that to work.
docker network inspect core-service-net >/dev/null 2>&1 || docker network create core-service-net

docker pull ${docker_image}
docker pull ${nginx_image}

docker rm -f nginx || true
docker rm -f core-service || true

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

# Bound to 127.0.0.1 only, for SSH-side debugging (curl from the instance
# itself) — not reachable from outside since it's not in the security group.
# The public entrypoint is nginx below.
docker run -d \
  --name core-service \
  --restart unless-stopped \
  --network core-service-net \
  -p 127.0.0.1:${app_port}:${app_port} \
  -v /opt/core-service/app.env:/app/app.env:ro \
  ${docker_image}

mkdir -p /opt/nginx
echo '${nginx_conf_base64}' | base64 -d > /opt/nginx/default.conf
chmod 644 /opt/nginx/default.conf

docker run -d \
  --name nginx \
  --restart unless-stopped \
  --network core-service-net \
  -p ${nginx_port}:${nginx_port} \
  -v /opt/nginx/default.conf:/etc/nginx/conf.d/default.conf:ro \
  ${nginx_image}
